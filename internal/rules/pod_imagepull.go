package rules

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"

	"github.com/khvedela/kubediag/internal/findings"
)

func init() {
	Register(&podImagePullBackOff{})
	Register(&podImageNotFound{})
	Register(&podImageAuth{})
}

// ----- TRG-POD-IMAGEPULLBACKOFF -------------------------------------------

type podImagePullBackOff struct{}

func (r *podImagePullBackOff) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-IMAGEPULLBACKOFF",
		Title:    "Container image pull is failing (ImagePullBackOff / ErrImagePull)",
		Category: findings.CategoryImage,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The kubelet cannot pull the container image. Kubernetes backs off retries
exponentially. The most common sub-causes are: wrong image name/tag, image not
found, or authentication failure against a private registry.

See TRG-POD-IMAGE-NOT-FOUND and TRG-POD-IMAGE-AUTH for specialised rules that
fire when the event message is more specific.`,
		DocsLinks: []string{"https://kubernetes.io/docs/concepts/containers/images/"},
		Priority:  85,
	}
}

func (r *podImagePullBackOff) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)

	var out []findings.Finding
	for _, cs := range allContainerStatuses(pod) {
		if !isImagePullState(cs) {
			continue
		}
		// If a more specific rule fired (auth / not-found) let that rule own it.
		msg := imagePullEventMsg(events, cs.Name)
		if isAuthError(msg) || isNotFoundError(msg) {
			continue
		}

		ev := imagePullEvidence(cs, msg)
		ns, name := rc.Target.Namespace, rc.Target.Name
		out = append(out, findings.Finding{
			ID:         "TRG-POD-IMAGEPULLBACKOFF",
			RuleID:     "TRG-POD-IMAGEPULLBACKOFF",
			Title:      fmt.Sprintf("Container %q cannot pull image %q", cs.Name, cs.Image),
			Summary:    fmt.Sprintf("Kubernetes cannot pull %q. Check the image name, tag, and whether the registry is reachable from the cluster.", cs.Image),
			Category:   findings.CategoryImage,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "Confirm the image name and tag exist. If the registry is private, ensure an imagePullSecret is configured.",
				NextCommands: []string{
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
					fmt.Sprintf("docker pull %s   # test from a node that can reach the registry", cs.Image),
				},
				SuggestedFix: "Verify the image exists at the exact tag. " +
					"If using a private registry, create an imagePullSecret and reference it in the pod spec.",
				DocsLinks: []string{"https://kubernetes.io/docs/concepts/containers/images/"},
			},
		})
	}
	return out, nil
}

// ----- TRG-POD-IMAGE-NOT-FOUND --------------------------------------------

type podImageNotFound struct{}

func (r *podImageNotFound) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-IMAGE-NOT-FOUND",
		Title:    "Container image tag or repository does not exist",
		Category: findings.CategoryImage,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The registry responded that the specified image or tag was not found.
This is almost always a typo in the image reference (wrong tag, wrong repo path,
deleted image) rather than a network or auth issue.`,
		DocsLinks: []string{"https://kubernetes.io/docs/concepts/containers/images/"},
		Priority:  88,
	}
}

func (r *podImageNotFound) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)

	var out []findings.Finding
	for _, cs := range allContainerStatuses(pod) {
		if !isImagePullState(cs) {
			continue
		}
		msg := imagePullEventMsg(events, cs.Name)
		if !isNotFoundError(msg) {
			continue
		}

		ns, name := rc.Target.Namespace, rc.Target.Name
		ev := imagePullEvidence(cs, msg)
		out = append(out, findings.Finding{
			ID:         "TRG-POD-IMAGE-NOT-FOUND",
			RuleID:     "TRG-POD-IMAGE-NOT-FOUND",
			Title:      fmt.Sprintf("Image %q not found in registry", cs.Image),
			Summary:    fmt.Sprintf("The registry reports that image %q does not exist. The tag may be misspelled or the image may have been deleted.", cs.Image),
			Category:   findings.CategoryImage,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: findings.Remediation{
				Explanation: "The image tag does not exist at this registry. Check for typos, deleted tags, or a wrong registry URL.",
				NextCommands: []string{
					fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				},
				SuggestedFix: fmt.Sprintf("Verify that %q exists in the registry. "+
					"List available tags: `docker search` or use the registry's UI/API.", cs.Image),
			},
		})
	}
	return out, nil
}

// ----- TRG-POD-IMAGE-AUTH -------------------------------------------------

type podImageAuth struct{}

func (r *podImageAuth) Meta() findings.RuleMeta {
	return findings.RuleMeta{
		ID:       "TRG-POD-IMAGE-AUTH",
		Title:    "Container image pull failed due to authentication/authorisation error",
		Category: findings.CategoryImage,
		Severity: findings.SeverityHigh,
		Scopes:   []findings.TargetKind{findings.TargetKindPod},
		Description: `The registry rejected the pull because credentials are missing, expired, or
incorrect. This can also mean the service account's imagePullSecret was not
configured, or the secret's credentials have rotated.`,
		DocsLinks: []string{
			"https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/",
		},
		Priority: 89,
	}
}

func (r *podImageAuth) Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error) {
	pod, found, err := rc.Cache.GetPod(ctx, rc.Target.Namespace, rc.Target.Name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	events, _ := rc.Cache.ListEventsFor(ctx, "Pod", rc.Target.Namespace, rc.Target.Name)

	var out []findings.Finding
	for _, cs := range allContainerStatuses(pod) {
		if !isImagePullState(cs) {
			continue
		}
		msg := imagePullEventMsg(events, cs.Name)
		if !isAuthError(msg) {
			continue
		}

		ns, name := rc.Target.Namespace, rc.Target.Name
		ev := imagePullEvidence(cs, msg)

		// Show which imagePullSecrets the pod is using.
		var secretNames []string
		for _, s := range pod.Spec.ImagePullSecrets {
			secretNames = append(secretNames, s.Name)
		}

		remedy := findings.Remediation{
			Explanation: "The registry rejected the pull due to authentication failure. " +
				"Ensure the imagePullSecret contains valid, non-expired credentials for this registry.",
			NextCommands: []string{
				fmt.Sprintf("kubectl describe pod -n %s %s", ns, name),
				fmt.Sprintf("kubectl get secret -n %s", ns),
			},
			SuggestedFix: "Create or refresh an imagePullSecret with valid registry credentials " +
				"and reference it in spec.imagePullSecrets. " +
				"If using a cloud registry (ECR, GCR, ACR), ensure the node's IAM role or workload identity is correctly configured.",
			DocsLinks: []string{
				"https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/",
			},
		}
		if len(secretNames) > 0 {
			remedy.NextCommands = append(remedy.NextCommands,
				fmt.Sprintf("kubectl get secret -n %s %s -o jsonpath='{.data.%s}' | base64 -d | jq .auths",
					ns, strings.Join(secretNames, ","), ".dockerconfigjson"))
		}

		out = append(out, findings.Finding{
			ID:         "TRG-POD-IMAGE-AUTH",
			RuleID:     "TRG-POD-IMAGE-AUTH",
			Title:      fmt.Sprintf("Image pull authentication failed for %q", cs.Image),
			Summary:    fmt.Sprintf("The registry rejected pull of %q with an auth error. Check imagePullSecrets and registry credentials.", cs.Image),
			Category:   findings.CategoryImage,
			Severity:   findings.SeverityHigh,
			Confidence: findings.ConfidenceHigh,
			Target:     rc.Target,
			Evidence:   ev,
			Remediation: remedy,
		})
	}
	return out, nil
}

// ----- shared helpers -------------------------------------------------------

func allContainerStatuses(pod *corev1.Pod) []corev1.ContainerStatus {
	out := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	out = append(out, pod.Status.ContainerStatuses...)
	out = append(out, pod.Status.InitContainerStatuses...)
	return out
}

func isImagePullState(cs corev1.ContainerStatus) bool {
	if cs.State.Waiting == nil {
		return false
	}
	r := cs.State.Waiting.Reason
	return r == "ImagePullBackOff" || r == "ErrImagePull"
}

// imagePullEventMsg returns the message from the most recent Failed/Warning
// event matching this container's image pull, or "" if none found.
func imagePullEventMsg(events []eventsv1.Event, containerName string) string {
	_ = containerName // events from involvedObject.name=podName already scoped
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if e.Type == "Warning" &&
			(e.Reason == "Failed" || e.Reason == "BackOff") &&
			strings.Contains(strings.ToLower(e.Note), "pull") {
			return e.Note
		}
	}
	return ""
}

func imagePullEvidence(cs corev1.ContainerStatus, eventMsg string) []findings.Evidence {
	ev := []findings.Evidence{
		{
			Kind:   findings.EvidenceKindField,
			Source: fmt.Sprintf("pod.status.containerStatuses[%s].state.waiting.reason", cs.Name),
			Value:  cs.State.Waiting.Reason,
		},
		{
			Kind:   findings.EvidenceKindField,
			Source: fmt.Sprintf("pod.status.containerStatuses[%s].image", cs.Name),
			Value:  cs.Image,
		},
	}
	if eventMsg != "" {
		ev = append(ev, findings.Evidence{
			Kind:  findings.EvidenceKindEvent,
			Value: eventMsg,
		})
	}
	return ev
}

func isAuthError(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "authentication required")
}

func isNotFoundError(msg string) bool {
	msg = strings.ToLower(msg)
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "manifest unknown") ||
		strings.Contains(msg, "does not exist")
}
