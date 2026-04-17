package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/rules"
)

func oomPod(ns, name, containerName string) *corev1.Pod {
	limit := resource.MustParse("256Mi")
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: containerName,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{corev1.ResourceMemory: limit},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         containerName,
					RestartCount: 2,
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
					},
				},
			},
		},
	}
}

func TestOOMKilled_Fires(t *testing.T) {
	pod := oomPod("default", "oom-pod", "app")
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-OOMKILLED")
	if r == nil {
		t.Fatal("rule TRG-POD-OOMKILLED not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding, got none")
	}
	f := out[0]
	if f.RuleID != "TRG-POD-OOMKILLED" {
		t.Errorf("wrong RuleID: %s", f.RuleID)
	}
	if f.Severity != findings.SeverityHigh {
		t.Errorf("want high severity, got %s", f.Severity)
	}
	if f.Confidence != findings.ConfidenceHigh {
		t.Errorf("want high confidence, got %s", f.Confidence)
	}
}

func TestOOMKilled_NoFire_WhenClean(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ok-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-OOMKILLED")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no findings for healthy pod, got %d", len(out))
	}
}

func TestOOMKilled_IncludesMemoryLimit(t *testing.T) {
	pod := oomPod("default", "oom-pod", "app")
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-OOMKILLED")
	out, _ := r.Evaluate(context.Background(), rc)
	if len(out) == 0 {
		t.Fatal("expected finding")
	}
	found := false
	for _, ev := range out[0].Evidence {
		if ev.Value == "256Mi" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected memory limit 256Mi in evidence")
	}
}
