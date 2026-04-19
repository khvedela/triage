package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/rules"
)

func TestInitFailed_Fires(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "init-fail", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "setup",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Error",
							ExitCode: 1,
						},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-INIT-FAILED")
	if r == nil {
		t.Fatal("rule not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding, got none")
	}
	if out[0].Severity != findings.SeverityHigh {
		t.Errorf("want high severity, got %s", out[0].Severity)
	}
}

func TestInitFailed_NoFire_WhenExitZero(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "init-ok", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "setup",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Completed",
							ExitCode: 0,
						},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-INIT-FAILED")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for exit 0; got %d findings", len(out))
	}
}

func TestInitFailed_NoFire_WhenCrashLoopBackOff(t *testing.T) {
	// CrashLoopBackOff init containers are handled by the crashloop rule, not this one.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "init-crash", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "setup",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-INIT-FAILED")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for CrashLoopBackOff; got %d findings", len(out))
	}
}
