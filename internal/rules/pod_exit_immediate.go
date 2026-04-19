package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/khvedela/kubediag/internal/findings"
)

func init() { Register(&podExitImmediate{}) }

// TRG-POD-EXIT-IMMEDIATE detects containers whose last termination indicates
// the binary itself could not execute: exit codes 126/127, or a termination
// message containing "exec format error" / "no such file or directory".
type podExitImmediate struct{}

func (r *podExitImmediate) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-EXIT-IMMEDIATE",
		Title:    "Container exits immediately — exec format error or missing binary",
		Category: findings.CategoryRuntime,
		Severity: findings.SeverityCritical,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The container process exited within seconds of starting with a non-zero exit
code that indicates the entrypoint binary could not be executed at all:

- Exit code 126: binary found but not executable (wrong permissions or file type)
- Exit code 127: binary not found in PATH (missing from image, wrong entrypoint)
- "exec format error": image built for a different CPU architecture (e.g. amd64 image on arm64 node)
- "no such file or directory": entrypoint path does not exist inside the container

These are distinct from application crashes because the process never started —
the kernel rejected the binary before any user code ran.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/",
			"https://docs.docker.com/build/building/multi-platform/",
		},
		Priority: 95,
	}
}

func (r *podExitImmediate) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil || !found {
		return nil, err
	}

	ns, name := rc.Target.Namespace, rc.Target.Name
	var out []findings.Finding

	allStatuses := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	for _, cs := range allStatuses {
		exitCode, message, detected := lastTerminatedExecFailure(cs)
		if !detected {
			continue
		}

		title, summary, explanation := classifyExecExit(cs.Name, exitCode, message)

		ev := []findings.Evidence{
			{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("pod.status.containerStatuses[%s].lastTerminationState.terminated.exitCode", cs.Name),
				Value:  fmt.Sprintf("%d", exitCode),
			},
		}
		if message != "" {
			ev = append(ev, findings.Evidence{
				Kind:   findings.EvidenceKindField,
				Source: "terminated.message",
				Value:  truncate(message, 200),
			})
		}
		if cs.Image != "" {
			ev = append(ev, findings.Evidence{
				Kind:   findings.EvidenceKindField,
				Source: fmt.Sprintf("pod.spec.containers[%s].image", cs.Name),
				Value:  cs.Image,
			})
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-EXIT-IMMEDIATE",
			RuleID:     "TRG-POD-EXIT-IMMEDIATE",
			Title:      title,
			Summary:    summary,
			Category:   findings.CategoryRuntime,
			Severity:   findings.SeverityCritical,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: explanation,
				NextCommands: []string{
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
					fmt.Sprintf("kubectl get pod -n %s %s -o jsonpath='{.status.containerStatuses}'", ns, name),
					fmt.Sprintf("docker run --rm --entrypoint sh %s -c 'uname -m'", cs.Image),
				},
				SuggestedFix: execExitFix(exitCode, message),
				DocsLinks: []string{
					"https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/",
					"https://docs.docker.com/build/building/multi-platform/",
				},
			},
		})
	}
	return out, nil
}

// lastTerminatedExecFailure inspects the container status and returns
// (exitCode, message, true) when the last termination looks like an
// exec-level failure rather than an application crash.
func lastTerminatedExecFailure(cs corev1.ContainerStatus) (int32, string, bool) {
	var exitCode int32
	var message string

	// Prefer LastTerminationState (catches CrashLoopBackOff pattern).
	if t := cs.LastTerminationState.Terminated; t != nil {
		exitCode = t.ExitCode
		message = t.Message
	} else if t := cs.State.Terminated; t != nil {
		exitCode = t.ExitCode
		message = t.Message
	} else {
		return 0, "", false
	}

	if isImmediateExitCode(exitCode) || isExecFormatMessage(message) {
		return exitCode, message, true
	}
	return 0, "", false
}

func isImmediateExitCode(code int32) bool {
	return code == 126 || code == 127
}

func isExecFormatMessage(msg string) bool {
	low := strings.ToLower(msg)
	return strings.Contains(low, "exec format error") ||
		strings.Contains(low, "no such file or directory") ||
		strings.Contains(low, "executable file not found") ||
		strings.Contains(low, "cannot execute binary file")
}

func classifyExecExit(container string, exitCode int32, message string) (title, summary, explanation string) {
	msgLow := strings.ToLower(message)
	switch {
	case strings.Contains(msgLow, "exec format error"):
		title = fmt.Sprintf("Container %q: exec format error — wrong CPU architecture", container)
		summary = fmt.Sprintf("Container %q failed to start: the binary cannot execute on this node's CPU architecture. "+
			"The image was likely built for a different platform (e.g. linux/amd64 on an arm64 node). Exit code: %d.", container, exitCode)
		explanation = "The Linux kernel rejected the binary because it was compiled for a different CPU architecture. " +
			"Build a multi-arch image that includes the variant matching this node."
	case exitCode == 127 || strings.Contains(msgLow, "no such file or directory") || strings.Contains(msgLow, "executable file not found"):
		title = fmt.Sprintf("Container %q: entrypoint binary not found (exit 127)", container)
		summary = fmt.Sprintf("Container %q could not start: its entrypoint command was not found inside the image. "+
			"The PATH may be wrong or the binary may be missing. Exit code: %d.", container, exitCode)
		explanation = "Exit code 127 means the container runtime or shell could not locate the command specified as ENTRYPOINT/CMD."
	case exitCode == 126:
		title = fmt.Sprintf("Container %q: entrypoint not executable (exit 126)", container)
		summary = fmt.Sprintf("Container %q could not start: the entrypoint binary exists but is not executable "+
			"(permission denied or wrong file type). Exit code: %d.", container, exitCode)
		explanation = "Exit code 126 means the binary was found but cannot be executed, usually missing execute permission or wrong file type."
	default:
		title = fmt.Sprintf("Container %q: immediate exec failure (exit code %d)", container, exitCode)
		summary = fmt.Sprintf("Container %q exited immediately with code %d: %s", container, exitCode, truncate(message, 100))
		explanation = "The container exited before any application code ran — the entrypoint binary could not be executed."
	}
	return
}

func execExitFix(exitCode int32, message string) string {
	msgLow := strings.ToLower(message)
	switch {
	case strings.Contains(msgLow, "exec format error"):
		return "Rebuild the image for the correct platform:\n" +
			"  docker buildx build --platform linux/amd64,linux/arm64 -t <image> --push .\n" +
			"Or add a nodeSelector to pin the pod to nodes matching the image architecture."
	case exitCode == 127:
		return "Verify ENTRYPOINT/CMD in the Dockerfile references a binary inside the image:\n" +
			"  docker run --rm --entrypoint sh <image> -c 'which <your-binary>'\n" +
			"Ensure the binary is installed and PATH is correct."
	case exitCode == 126:
		return "Add execute permission in the Dockerfile:\n" +
			"  RUN chmod +x /entrypoint\n" +
			"Verify the file is a real ELF executable or shell script."
	default:
		return "Inspect the container image entrypoint and verify it exists and is executable for the target architecture."
	}
}
