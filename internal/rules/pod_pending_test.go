package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
	"github.com/OWNER/triage/internal/rules"
)

func pendingPodWithEvent(ns, name, eventMsg string) (*corev1.Pod, eventsv1.Event) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("2"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}
	ev := eventsv1.Event{
		Type:   "Warning",
		Reason: "FailedScheduling",
		Note:   eventMsg,
	}
	return pod, ev
}

func newPendingRC(t *testing.T, pod *corev1.Pod, ev eventsv1.Event) *rules.Context {
	t.Helper()
	fc := kube.NewFakeClient()
	fc.AddPod(pod)
	fc.AddEventFor("Pod", pod.Namespace, pod.Name, ev)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: pod.Namespace, Name: pod.Name},
		Cache:  cache,
	}
}

func TestPendingResources_Fires(t *testing.T) {
	pod, ev := pendingPodWithEvent("default", "big-pod", "0/3 nodes available: 3 Insufficient memory")
	rc := newPendingRC(t, pod, ev)
	r := rules.Get("TRG-POD-PENDING-INSUFFICIENT-RESOURCES")
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
	if out[0].RuleID != "TRG-POD-PENDING-INSUFFICIENT-RESOURCES" {
		t.Errorf("wrong ruleID: %s", out[0].RuleID)
	}
}

func TestPendingTaint_Fires(t *testing.T) {
	pod, ev := pendingPodWithEvent("default", "taint-pod", "0/3 nodes available: 3 node(s) had untolerated taint {dedicated: gpu}")
	rc := newPendingRC(t, pod, ev)
	r := rules.Get("TRG-POD-PENDING-TAINT-MISMATCH")
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
}

func TestPendingSelector_Fires(t *testing.T) {
	pod, ev := pendingPodWithEvent("default", "sel-pod", "0/3 nodes available: 3 node(s) didn't match Pod's node affinity/selector")
	rc := newPendingRC(t, pod, ev)
	r := rules.Get("TRG-POD-PENDING-SELECTOR-MISMATCH")
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
}

func TestPendingResources_NoFire_WhenRunning(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "running", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	rc := newPodRC(t, pod)
	r := rules.Get("TRG-POD-PENDING-INSUFFICIENT-RESOURCES")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for running pod; got %d findings", len(out))
	}
}

func TestPendingPVC_Fires_WhenPVCNotBound(t *testing.T) {
	sc := "standard"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "default"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &sc},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "my-pvc"},
				}},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}

	fc := kube.NewFakeClient()
	fc.AddPod(pod)
	fc.AddPVC(pvc)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: "default", Name: "pvc-pod"},
		Cache:  cache,
	}

	r := rules.Get("TRG-POD-PENDING-PVC-UNBOUND")
	if r == nil {
		t.Fatal("rule not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding for unbound PVC, got none")
	}
}

func TestPendingPVC_NoFire_WhenPVCBound(t *testing.T) {
	sc := "standard"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvc", Namespace: "default"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &sc},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "my-pvc"},
				}},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	fc := kube.NewFakeClient()
	fc.AddPod(pod)
	fc.AddPVC(pvc)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: "default", Name: "pvc-pod"},
		Cache:  cache,
	}

	r := rules.Get("TRG-POD-PENDING-PVC-UNBOUND")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire when PVC is Bound; got %d findings", len(out))
	}
}
