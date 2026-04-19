package rules_test

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/kube"
	"github.com/khvedela/triage/internal/rules"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newNSRC(t *testing.T, ns string, fc *kube.FakeClient) *rules.Context {
	t.Helper()
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	return &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindNamespace, Namespace: ns, Name: ns},
		Cache:  cache,
		Now:    time.Now,
	}
}

func mustRule(t *testing.T, id string) rules.Rule {
	t.Helper()
	r := rules.Get(id)
	if r == nil {
		t.Fatalf("rule %s not registered", id)
	}
	return r
}

func assertFindings(t *testing.T, out []findings.Finding, err error, wantCount int, ruleID string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != wantCount {
		t.Fatalf("want %d finding(s), got %d", wantCount, len(out))
	}
	if wantCount > 0 && out[0].RuleID != ruleID {
		t.Errorf("want RuleID %s, got %s", ruleID, out[0].RuleID)
	}
}

// ─── TRG-POD-EXIT-IMMEDIATE ───────────────────────────────────────────────────

func TestExitImmediate_ExitCode127_Fires(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "missing-bin", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "busybox:latest"}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Image: "busybox:latest",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 127,
							Message:  "executable file not found in $PATH",
						},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := mustRule(t, "TRG-POD-EXIT-IMMEDIATE")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-EXIT-IMMEDIATE")
	if out[0].Severity != findings.SeverityCritical {
		t.Errorf("want critical, got %s", out[0].Severity)
	}
}

func TestExitImmediate_ExecFormatError_Fires(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "wrong-arch", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "myapp:amd64"}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Image: "myapp:amd64",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1,
							Message:  "exec format error",
						},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := mustRule(t, "TRG-POD-EXIT-IMMEDIATE")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-EXIT-IMMEDIATE")
}

func TestExitImmediate_ExitCode126_Fires(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "no-exec", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "myapp:v1"}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{ExitCode: 126},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := mustRule(t, "TRG-POD-EXIT-IMMEDIATE")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-EXIT-IMMEDIATE")
}

func TestExitImmediate_NormalCrash_NoFire(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "normal-crash", Namespace: "default"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Message: "application error"},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := mustRule(t, "TRG-POD-EXIT-IMMEDIATE")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

func TestExitImmediate_InitContainer_Fires(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-init", Namespace: "default"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "init", Image: "init:v1"}},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "init",
					LastTerminationState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{ExitCode: 127},
					},
				},
			},
		},
	}
	rc := newPodRC(t, pod)
	r := mustRule(t, "TRG-POD-EXIT-IMMEDIATE")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-EXIT-IMMEDIATE")
}

// ─── TRG-SVC-PORT-MISMATCH ────────────────────────────────────────────────────

func mismatchSvcSetup(t *testing.T) (*kube.FakeClient, string) {
	t.Helper()
	fc := kube.NewFakeClient()
	ns := "web"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns, Labels: map[string]string{"app": "web"}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "server",
					Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
				},
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt(9090)},
			},
		},
	}
	fc.AddPod(pod)
	fc.AddService(svc)
	return fc, ns
}

func TestSvcPortMismatch_Fires(t *testing.T) {
	fc, ns := mismatchSvcSetup(t)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-SVC-PORT-MISMATCH")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-SVC-PORT-MISMATCH")
}

func TestSvcPortMismatch_NoFire_WhenPortMatches(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "web"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns, Labels: map[string]string{"app": "web"}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Ports: []corev1.ContainerPort{{ContainerPort: 8080}}},
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}},
		},
	}
	fc.AddPod(pod)
	fc.AddService(svc)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-SVC-PORT-MISMATCH")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

func TestSvcPortMismatch_NoFire_WhenNoPodsDeclaredPorts(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "web"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns, Labels: map[string]string{"app": "web"}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "server"}}}, // no ports declared
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(9090)}},
		},
	}
	fc.AddPod(pod)
	fc.AddService(svc)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-SVC-PORT-MISMATCH")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "") // no containerPorts declared — skip
}

func TestSvcPortMismatch_NamedPort_Matches(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "api"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: ns, Labels: map[string]string{"app": "api"}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}}},
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "api-svc", Namespace: ns},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "api"},
			Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromString("http")}},
		},
	}
	fc.AddPod(pod)
	fc.AddService(svc)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-SVC-PORT-MISMATCH")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "") // named port matches — no finding
}

// ─── TRG-POD-BAD-ENV-REF ──────────────────────────────────────────────────────

func optBool(b bool) *bool { return &b }

func TestBadEnvRef_ConfigMapMissingKey_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "default"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: ns},
		Data:       map[string]string{"existing-key": "value"},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "server",
					Env: []corev1.EnvVar{
						{
							Name: "DB_HOST",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
									Key:                  "missing-key",
								},
							},
						},
					},
				},
			},
		},
	}
	fc.AddConfigMap(cm)
	fc.AddPod(pod)
	rc := newPodRC(t, pod)
	// rebuild cache with the configmap in it
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc.Cache = cache

	r := mustRule(t, "TRG-POD-BAD-ENV-REF")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-BAD-ENV-REF")
	if out[0].Confidence != findings.ConfidenceHigh {
		t.Errorf("want high confidence, got %s", out[0].Confidence)
	}
}

func TestBadEnvRef_SecretMissingKey_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "default"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: ns},
		Data:       map[string][]byte{"password": []byte("hunter2")},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "server",
					Env: []corev1.EnvVar{
						{
							Name: "API_TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "app-secret"},
									Key:                  "token", // does not exist
								},
							},
						},
					},
				},
			},
		},
	}
	fc.AddSecret(secret)
	fc.AddPod(pod)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: ns, Name: pod.Name},
		Cache:  cache,
		Now:    time.Now,
	}
	r := mustRule(t, "TRG-POD-BAD-ENV-REF")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-POD-BAD-ENV-REF")
}

func TestBadEnvRef_KeyExists_NoFire(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "default"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: ns},
		Data:       map[string]string{"db-host": "postgres.svc"},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "server",
					Env: []corev1.EnvVar{
						{
							Name: "DB_HOST",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
									Key:                  "db-host",
								},
							},
						},
					},
				},
			},
		},
	}
	fc.AddConfigMap(cm)
	fc.AddPod(pod)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: ns, Name: pod.Name},
		Cache:  cache,
		Now:    time.Now,
	}
	r := mustRule(t, "TRG-POD-BAD-ENV-REF")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

func TestBadEnvRef_Optional_NoFire(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "default"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: ns},
		Data:       map[string]string{},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "server",
					Env: []corev1.EnvVar{
						{
							Name: "OPTIONAL_VAR",
							ValueFrom: &corev1.EnvVarSource{
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
									Key:                  "optional-key",
									Optional:             optBool(true),
								},
							},
						},
					},
				},
			},
		},
	}
	fc.AddConfigMap(cm)
	fc.AddPod(pod)
	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindPod, Namespace: ns, Name: pod.Name},
		Cache:  cache,
		Now:    time.Now,
	}
	r := mustRule(t, "TRG-POD-BAD-ENV-REF")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

// ─── TRG-CLUSTER-QUOTA-EXHAUSTED ──────────────────────────────────────────────

func TestQuotaExhausted_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "production"
	q := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "compute", Namespace: ns},
		Status: corev1.ResourceQuotaStatus{
			Hard: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("10"),
			},
			Used: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("10"),
			},
		},
	}
	fc.AddResourceQuota(q)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-CLUSTER-QUOTA-EXHAUSTED")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-CLUSTER-QUOTA-EXHAUSTED")
	if out[0].Severity != findings.SeverityCritical {
		t.Errorf("want critical, got %s", out[0].Severity)
	}
}

func TestQuotaExhausted_NearlyFull_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "staging"
	q := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "compute", Namespace: ns},
		Status: corev1.ResourceQuotaStatus{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100"),
			},
			Used: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("97"),
			},
		},
	}
	fc.AddResourceQuota(q)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-CLUSTER-QUOTA-EXHAUSTED")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-CLUSTER-QUOTA-EXHAUSTED")
	if out[0].Severity != findings.SeverityHigh {
		t.Errorf("want high, got %s", out[0].Severity)
	}
}

func TestQuotaExhausted_LowUsage_NoFire(t *testing.T) {
	fc := kube.NewFakeClient()
	ns := "dev"
	q := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Name: "compute", Namespace: ns},
		Status: corev1.ResourceQuotaStatus{
			Hard: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("100"),
			},
			Used: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("20"),
			},
		},
	}
	fc.AddResourceQuota(q)
	rc := newNSRC(t, ns, fc)
	r := mustRule(t, "TRG-CLUSTER-QUOTA-EXHAUSTED")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

// ─── TRG-CLUSTER-APISERVER-LATENCY ────────────────────────────────────────────

func TestAPIServerLatency_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ev := eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "ev1", Namespace: "kube-system"},
		Type:       "Warning",
		Reason:     "SlowReadResponse",
		Note:       "Read from etcd took 3.2s, exceeding the slowReadThreshold",
	}
	fc.AddEventFor("Node", "kube-system", "control-plane", ev)

	cache := kube.NewResourceCache(fc, kube.DiscardLogger())
	rc := &rules.Context{
		Target: findings.Target{Kind: findings.TargetKindCluster},
		Cache:  cache,
		Now:    time.Now,
	}
	r := mustRule(t, "TRG-CLUSTER-APISERVER-LATENCY")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-CLUSTER-APISERVER-LATENCY")
}

func TestAPIServerLatency_NormalEvents_NoFire(t *testing.T) {
	fc := kube.NewFakeClient()
	ev := eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "ev1", Namespace: "kube-system"},
		Type:       "Normal",
		Reason:     "Scheduled",
		Note:       "Successfully assigned pod to node",
	}
	fc.AddEventFor("Pod", "kube-system", "some-pod", ev)
	rc := newClusterRC(t, fc)
	r := mustRule(t, "TRG-CLUSTER-APISERVER-LATENCY")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 0, "")
}

func TestAPIServerLatency_EtcdTimeout_Fires(t *testing.T) {
	fc := kube.NewFakeClient()
	ev := eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "ev1", Namespace: "kube-system"},
		Type:       "Warning",
		Reason:     "BackoffLimitExceeded",
		Note:       "etcd cluster is unavailable: context deadline exceeded",
	}
	fc.AddEventFor("Pod", "kube-system", "etcd-0", ev)
	rc := newClusterRC(t, fc)
	r := mustRule(t, "TRG-CLUSTER-APISERVER-LATENCY")
	out, err := r.Evaluate(context.Background(), rc)
	assertFindings(t, out, err, 1, "TRG-CLUSTER-APISERVER-LATENCY")
}
