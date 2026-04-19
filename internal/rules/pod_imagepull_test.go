package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/kube"
	"github.com/khvedela/kubediag/internal/rules"
)

func imagePullPod(ns, name, image, waitingReason string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Image: image,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: waitingReason},
					},
				},
			},
		},
	}
}

func newPodRCWithEvents(t *testing.T, pod *corev1.Pod, events []eventsv1.Event) *rules.Context {
	t.Helper()
	fc := kube.NewFakeClient()
	fc.AddPod(pod)
	for _, ev := range events {
		fc.AddEventFor("Pod", pod.Namespace, pod.Name, ev)
	}
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: pod.Namespace, Name: pod.Name},
		Cache:  cache,
	}
}

func TestImagePullBackOff_Fires_Generic(t *testing.T) {
	pod := imagePullPod("default", "ip-pod", "myrepo/myimage:v1", "ImagePullBackOff")
	rc := newPodRCWithEvents(t, pod, nil)
	r := rules.Get("TRG-POD-IMAGEPULLBACKOFF")
	if r == nil {
		t.Fatal("rule TRG-POD-IMAGEPULLBACKOFF not registered")
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

func TestImageNotFound_Fires_WhenEventSaysNotFound(t *testing.T) {
	pod := imagePullPod("default", "ip-pod", "myrepo/myimage:bad-tag", "ImagePullBackOff")
	events := []eventsv1.Event{
		{
			Type:   "Warning",
			Reason: "Failed",
			Note:   "Failed to pull image: manifest for myrepo/myimage:bad-tag not found",
		},
	}
	rc := newPodRCWithEvents(t, pod, events)
	r := rules.Get("TRG-POD-IMAGE-NOT-FOUND")
	if r == nil {
		t.Fatal("rule TRG-POD-IMAGE-NOT-FOUND not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding, got none")
	}
	if out[0].RuleID != "TRG-POD-IMAGE-NOT-FOUND" {
		t.Errorf("wrong rule ID: %s", out[0].RuleID)
	}
}

func TestImageAuth_Fires_WhenEventSaysUnauthorized(t *testing.T) {
	pod := imagePullPod("default", "ip-pod", "private.registry.io/img:latest", "ImagePullBackOff")
	events := []eventsv1.Event{
		{
			Type:   "Warning",
			Reason: "Failed",
			Note:   "Failed to pull image: unauthorized: authentication required",
		},
	}
	rc := newPodRCWithEvents(t, pod, events)
	r := rules.Get("TRG-POD-IMAGE-AUTH")
	if r == nil {
		t.Fatal("rule TRG-POD-IMAGE-AUTH not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding, got none")
	}
}

func TestImagePullBackOff_NoFire_WhenAuthEventPresent(t *testing.T) {
	// The generic imagepull rule should defer to the auth rule when auth error is in event.
	pod := imagePullPod("default", "ip-pod", "private.registry.io/img:latest", "ImagePullBackOff")
	events := []eventsv1.Event{
		{
			Type:   "Warning",
			Reason: "Failed",
			Note:   "Failed to pull image: unauthorized: access denied",
		},
	}
	rc := newPodRCWithEvents(t, pod, events)
	r := rules.Get("TRG-POD-IMAGEPULLBACKOFF")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("generic rule should not fire when auth error is present; got %d findings", len(out))
	}
}

func TestImagePullBackOff_NoFire_WhenRunning(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "running-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}
	rc := newPodRCWithEvents(t, pod, nil)
	r := rules.Get("TRG-POD-IMAGEPULLBACKOFF")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected no findings for running pod, got %d", len(out))
	}
}
