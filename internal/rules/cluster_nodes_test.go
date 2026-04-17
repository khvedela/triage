package rules_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
	"github.com/OWNER/triage/internal/rules"
)

func newClusterRC(t *testing.T, fc *kube.FakeClient) *rules.Context {
	t.Helper()
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindCluster},
		Cache:  cache,
	}
}

func readyNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func notReadyNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse, Reason: "KubeletNotReady", Message: "PLEG is not healthy"},
			},
		},
	}
}

func pressureNode(name string, condType corev1.NodeConditionType) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: condType, Status: corev1.ConditionTrue, Message: "low memory"},
			},
		},
	}
}

func TestNodeNotReady_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddNode(readyNode("node-1"))
	fc.AddNode(notReadyNode("node-2"))

	rc := newClusterRC(t, fc)
	r := rules.Get("TRG-CLUSTER-NODE-NOT-READY")
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

func TestNodeNotReady_NoFire_WhenAllReady(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddNode(readyNode("node-1"))
	fc.AddNode(readyNode("node-2"))

	rc := newClusterRC(t, fc)
	r := rules.Get("TRG-CLUSTER-NODE-NOT-READY")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire when all nodes ready; got %d findings", len(out))
	}
}

func TestNodeNotReady_Critical_WhenMajorityDown(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddNode(readyNode("node-1"))
	fc.AddNode(notReadyNode("node-2"))
	fc.AddNode(notReadyNode("node-3"))

	rc := newClusterRC(t, fc)
	r := rules.Get("TRG-CLUSTER-NODE-NOT-READY")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding")
	}
	if out[0].Severity != findings.SeverityCritical {
		t.Errorf("want critical when majority down, got %s", out[0].Severity)
	}
}

func TestNodePressure_Fires_MemoryPressure(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddNode(pressureNode("node-1", corev1.NodeMemoryPressure))

	rc := newClusterRC(t, fc)
	r := rules.Get("TRG-CLUSTER-NODE-PRESSURE")
	if r == nil {
		t.Fatal("rule not registered")
	}
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected finding for memory pressure, got none")
	}
}

func TestNodePressure_NoFire_WhenNoPressure(t *testing.T) {
	fc := kube.NewFakeClient()
	fc.AddNode(readyNode("node-1"))

	rc := newClusterRC(t, fc)
	r := rules.Get("TRG-CLUSTER-NODE-PRESSURE")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire when no pressure; got %d findings", len(out))
	}
}
