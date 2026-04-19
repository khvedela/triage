package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/kube"
	"github.com/khvedela/kubediag/internal/rules"
)

func newNsRC(t *testing.T, ns string, fc *kube.FakeClient) *rules.Context {
	t.Helper()
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindNamespace, Namespace: ns, Name: ns},
		Cache:  cache,
	}
}

func svcWithSelector(ns, name string, selector map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1.ServiceSpec{Selector: selector},
	}
}

func emptyEndpoints(ns, name string) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Subsets:    nil,
	}
}

func populatedEndpoints(ns, name string) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Subsets: []corev1.EndpointSubset{
			{Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1"}}},
		},
	}
}

func TestSvcNoEndpoints_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddService(svcWithSelector("default", "my-svc", map[string]string{"app": "web"}))
	fc.AddEndpoints(emptyEndpoints("default", "my-svc"))

	rc := newNsRC(t, "default", fc)
	r := rules.Get("TRG-SVC-NO-ENDPOINTS")
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
	if out[0].RuleID != "TRG-SVC-NO-ENDPOINTS" {
		t.Errorf("wrong ruleID: %s", out[0].RuleID)
	}
}

func TestSvcNoEndpoints_NoFire_WhenEndpointsExist(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddService(svcWithSelector("default", "my-svc", map[string]string{"app": "web"}))
	fc.AddEndpoints(populatedEndpoints("default", "my-svc"))

	rc := newNsRC(t, "default", fc)
	r := rules.Get("TRG-SVC-NO-ENDPOINTS")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire when endpoints exist; got %d findings", len(out))
	}
}

func TestSvcNoEndpoints_NoFire_WhenExternalName(t *testing.T) {
	fc := kube.NewFakeClient()
	svc := svcWithSelector("default", "ext-svc", nil)
	svc.Spec.Type = corev1.ServiceTypeExternalName
	fc.AddService(svc)

	rc := newNsRC(t, "default", fc)
	r := rules.Get("TRG-SVC-NO-ENDPOINTS")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for ExternalName service; got %d findings", len(out))
	}
}

func TestSvcSelectorMismatch_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddService(svcWithSelector("default", "my-svc", map[string]string{"app": "web"}))
	fc.AddEndpoints(emptyEndpoints("default", "my-svc"))
	// Pod with different labels
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "api"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	fc.AddPod(pod)

	rc := newNsRC(t, "default", fc)
	r := rules.Get("TRG-SVC-SELECTOR-MISMATCH")
	if r == nil {
		t.Fatal("rule not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding for selector mismatch, got none")
	}
}

func TestSvcSelectorMismatch_NoFire_WhenPodMatches(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddService(svcWithSelector("default", "my-svc", map[string]string{"app": "web"}))
	fc.AddEndpoints(populatedEndpoints("default", "my-svc"))
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "web"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	fc.AddPod(pod)

	rc := newNsRC(t, "default", fc)
	r := rules.Get("TRG-SVC-SELECTOR-MISMATCH")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire when pod matches selector; got %d findings", len(out))
	}
}
