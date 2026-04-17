package rules_test

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/OWNER/triage/internal/findings"
	"github.com/OWNER/triage/internal/kube"
	"github.com/OWNER/triage/internal/rules"
)

func newDeployRC(t *testing.T, d *appsv1.Deployment) *rules.Context {
	t.Helper()
	fc := kube.NewFakeClient()
	fc.AddDeployment(d)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindDeployment, Namespace: d.Namespace, Name: d.Name},
		Cache:  cache,
	}
}

func stuckDeployment(ns, name string) *appsv1.Deployment {
	replicas := int32(3)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:       0,
			UnavailableReplicas: 3,
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionFalse,
					Reason: "ProgressDeadlineExceeded",
				},
			},
		},
	}
}

func TestDeployRolloutStuck_Fires(t *testing.T) {
	d := stuckDeployment("default", "my-deploy")
	rc := newDeployRC(t, d)
	r := rules.Get("TRG-DEPLOY-ROLLOUT-STUCK")
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
	if out[0].Severity != findings.SeverityCritical {
		t.Errorf("want critical severity, got %s", out[0].Severity)
	}
}

func TestDeployRolloutStuck_NoFire_WhenProgressing(t *testing.T) {
	replicas := int32(3)
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "ok-deploy", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionTrue,
					Reason: "NewReplicaSetAvailable",
				},
			},
		},
	}
	rc := newDeployRC(t, d)
	r := rules.Get("TRG-DEPLOY-ROLLOUT-STUCK")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not fire for healthy deploy; got %d findings", len(out))
	}
}

func TestDeployUnavailable_Fires(t *testing.T) {
	replicas := int32(3)
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "partial-deploy", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:       1,
			UnavailableReplicas: 2,
		},
	}
	rc := newDeployRC(t, d)
	r := rules.Get("TRG-DEPLOY-UNAVAILABLE-REPLICAS")
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

func TestDeployUnavailable_NoFire_WhenRolloutStuck(t *testing.T) {
	// When rollout-stuck already covers it, unavailable should defer.
	d := stuckDeployment("default", "stuck")
	rc := newDeployRC(t, d)
	r := rules.Get("TRG-DEPLOY-UNAVAILABLE-REPLICAS")
	out, err := r.Evaluate(context.Background(), rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("should not double-fire with rollout-stuck; got %d findings", len(out))
	}
}
