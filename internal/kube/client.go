package kube

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// Interface is the narrow Kubernetes surface triage uses. It is small by
// design: only the reads that rules actually need. A fake implementation for
// tests lives in internal/kube/fake.
type Interface interface {
	// Single-object reads
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error)
	GetReplicaSet(ctx context.Context, namespace, name string) (*appsv1.ReplicaSet, error)
	GetService(ctx context.Context, namespace, name string) (*corev1.Service, error)
	GetEndpoints(ctx context.Context, namespace, name string) (*corev1.Endpoints, error)
	GetConfigMap(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error)
	GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error)
	GetPVC(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error)
	GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error)
	GetNode(ctx context.Context, name string) (*corev1.Node, error)

	// List calls
	ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error)
	ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
	ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error)
	ListNodes(ctx context.Context) ([]corev1.Node, error)
	ListEventsFor(ctx context.Context, kind, namespace, name string) ([]eventsv1.Event, error)
	ListEventsInNamespace(ctx context.Context, namespace string) ([]eventsv1.Event, error)

	// Logs (lazy, per-container, tail-bounded)
	GetLogs(ctx context.Context, namespace, pod, container string, tailLines int64, previous bool) (string, error)

	// RBAC probe
	CanI(ctx context.Context, verb, group, resource, namespace string) (bool, error)

	// Context info
	CurrentNamespace() string
}

// NewClient constructs a Kubernetes client from the given cli-runtime config
// flags (--context, --kubeconfig, etc).
func NewClient(flags *genericclioptions.ConfigFlags) (Interface, error) {
	if flags == nil {
		flags = genericclioptions.NewConfigFlags(true)
	}
	restCfg, err := flags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}
	restCfg.UserAgent = "triage"
	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("build clientset: %w", err)
	}

	ns, _, _ := flags.ToRawKubeConfigLoader().Namespace()

	return &clientgoClient{cs: cs, namespace: ns}, nil
}

// clientgoClient is the production Interface backed by client-go.
type clientgoClient struct {
	cs        kubernetes.Interface
	namespace string
}

func (c *clientgoClient) CurrentNamespace() string {
	if c.namespace == "" {
		return "default"
	}
	return c.namespace
}

func (c *clientgoClient) GetPod(ctx context.Context, ns, name string) (*corev1.Pod, error) {
	return c.cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetDeployment(ctx context.Context, ns, name string) (*appsv1.Deployment, error) {
	return c.cs.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetReplicaSet(ctx context.Context, ns, name string) (*appsv1.ReplicaSet, error) {
	return c.cs.AppsV1().ReplicaSets(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetService(ctx context.Context, ns, name string) (*corev1.Service, error) {
	return c.cs.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetEndpoints(ctx context.Context, ns, name string) (*corev1.Endpoints, error) {
	return c.cs.CoreV1().Endpoints(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetConfigMap(ctx context.Context, ns, name string) (*corev1.ConfigMap, error) {
	return c.cs.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetSecret(ctx context.Context, ns, name string) (*corev1.Secret, error) {
	return c.cs.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetPVC(ctx context.Context, ns, name string) (*corev1.PersistentVolumeClaim, error) {
	return c.cs.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (c *clientgoClient) ListPods(ctx context.Context, ns string) ([]corev1.Pod, error) {
	l, err := c.cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListDeployments(ctx context.Context, ns string) ([]appsv1.Deployment, error) {
	l, err := c.cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListReplicaSets(ctx context.Context, ns string) ([]appsv1.ReplicaSet, error) {
	l, err := c.cs.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListServices(ctx context.Context, ns string) ([]corev1.Service, error) {
	l, err := c.cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListNetworkPolicies(ctx context.Context, ns string) ([]networkingv1.NetworkPolicy, error) {
	l, err := c.cs.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	l, err := c.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListEventsFor(ctx context.Context, kind, ns, name string) ([]eventsv1.Event, error) {
	selector := fmt.Sprintf("regarding.kind=%s,regarding.name=%s", kind, name)
	l, err := c.cs.EventsV1().Events(ns).List(ctx, metav1.ListOptions{FieldSelector: selector})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) ListEventsInNamespace(ctx context.Context, ns string) ([]eventsv1.Event, error) {
	l, err := c.cs.EventsV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientgoClient) GetLogs(ctx context.Context, ns, pod, container string, tailLines int64, previous bool) (string, error) {
	opts := &corev1.PodLogOptions{Container: container, Previous: previous, TailLines: &tailLines}
	req := c.cs.CoreV1().Pods(ns).GetLogs(pod, opts)
	data, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
