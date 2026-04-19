package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/kube"
	"github.com/khvedela/kubediag/internal/rules"
)

func TestNsWarningEvents_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddEventFor("Pod", "default", "some-pod", eventsv1.Event{
		Type:   "Warning",
		Reason: "BackOff",
		Note:   "Back-off restarting failed container",
		Regarding: corev1.ObjectReference{Kind: "Pod", Name: "some-pod"},
	})
	fc.AddEventFor("Pod", "default", "another-pod", eventsv1.Event{
		Type:   "Warning",
		Reason: "BackOff",
		Note:   "Back-off restarting failed container",
		Regarding: corev1.ObjectReference{Kind: "Pod", Name: "another-pod"},
	})

	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindNamespace, Name: "default", Namespace: "default"},
		Cache:  cache,
	}

	r := rules.Get("TRG-NS-WARNING-EVENTS")
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
	if out[0].Severity != findings.SeverityMedium {
		t.Errorf("want medium severity, got %s", out[0].Severity)
	}
}

func TestNsWarningEvents_NoFire_WhenNoWarnings(t *testing.T) {
	fc := kube.NewFakeClient()
	// Only Normal events — no warnings.
	fc.AddEventFor("Pod", "default", "ok-pod", eventsv1.Event{
		Type:   "Normal",
		Reason: "Started",
		Note:   "Container started",
	})

	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindNamespace, Name: "default", Namespace: "default"},
		Cache:  cache,
	}

	r := rules.Get("TRG-NS-WARNING-EVENTS")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for Normal events; got %d findings", len(out))
	}
}

func TestNsWarningEvents_NoFire_WhenWrongScope(t *testing.T) {
	fc := kube.NewFakeClient()
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Name: "my-pod", Namespace: "default"},
		Cache:  cache,
	}
	r := rules.Get("TRG-NS-WARNING-EVENTS")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for pod scope; got %d findings", len(out))
	}
}
