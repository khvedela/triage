package rules_test

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/kube"
	"github.com/khvedela/triage/internal/rules"
)

// newPodRC builds a minimal RuleContext backed by an in-memory fake client
// populated with the given pod.
func newPodRC(t *testing.T, pod *corev1.Pod) *rules.Context {
	t.Helper()
	fc := kube.NewFakeClient()
	fc.AddPod(pod)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: pod.Namespace, Name: pod.Name},
		Cache:  cache,
		Now:    time.Now,
	}
}

func crashLoopPod(ns, name, containerName string, restarts int32) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         containerName,
					RestartCount: restarts,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Error",
							ExitCode: 1,
						},
					},
				},
			},
		},
	}
}

func TestCrashLoopBackOff_Fires(t *testing.T) {
	pod := crashLoopPod("default", "my-pod", "app", 5)
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-CRASHLOOPBACKOFF")
	if r == nil {
		t.Fatal("rule TRG-POD-CRASHLOOPBACKOFF not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected findings, got none")
	}
	f := out[0]
	if f.RuleID != "TRG-POD-CRASHLOOPBACKOFF" {
		t.Errorf("wrong RuleID: %s", f.RuleID)
	}
	if f.Severity != findings.SeverityCritical {
		t.Errorf("want critical severity, got %s", f.Severity)
	}
	if f.Confidence != findings.ConfidenceHigh {
		t.Errorf("want high confidence, got %s", f.Confidence)
	}
}

func TestCrashLoopBackOff_NoFire_WhenRunningAndReady(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ok-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-CRASHLOOPBACKOFF")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no findings for healthy pod, got %d", len(out))
	}
}

func TestCrashLoopBackOff_NoFire_WhenPodNotFound(t *testing.T) {
	fc := kube.NewFakeClient()
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: "default", Name: "missing"},
		Cache:  cache,
		Now:    time.Now,
	}
	r := rules.Get("TRG-POD-CRASHLOOPBACKOFF")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no findings for missing pod, got %d", len(out))
	}
}

func TestCrashLoopBackOff_InitContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-init", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "init-setup",
					RestartCount: 3,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-CRASHLOOPBACKOFF")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding for init container CrashLoopBackOff")
	}
}
