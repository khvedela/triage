package kube

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/go-logr/logr"
)

// FakeClient implements Interface for use in unit tests. Populate it with
// AddPod, AddDeployment, etc. before running rules.
type FakeClient struct {
	pods          map[string]*corev1.Pod
	deployments   map[string]*appsv1.Deployment
	replicaSets   map[string]*appsv1.ReplicaSet
	services      map[string]*corev1.Service
	endpoints     map[string]*corev1.Endpoints
	configMaps    map[string]*corev1.ConfigMap
	secrets       map[string]*corev1.Secret
	pvcs          map[string]*corev1.PersistentVolumeClaim
	namespaces    map[string]*corev1.Namespace
	nodes         map[string]*corev1.Node
	events        map[string][]eventsv1.Event // key = "kind|ns|name"
	nsEvents      map[string][]eventsv1.Event // key = namespace
	netpols       map[string][]networkingv1.NetworkPolicy
	logs          map[string]string
	namespace     string

	// ForbiddenKinds is a set of resource kinds that return Forbidden.
	ForbiddenKinds map[string]bool
}

// NewFakeClient creates an empty FakeClient.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		pods:           map[string]*corev1.Pod{},
		deployments:    map[string]*appsv1.Deployment{},
		replicaSets:    map[string]*appsv1.ReplicaSet{},
		services:       map[string]*corev1.Service{},
		endpoints:      map[string]*corev1.Endpoints{},
		configMaps:     map[string]*corev1.ConfigMap{},
		secrets:        map[string]*corev1.Secret{},
		pvcs:           map[string]*corev1.PersistentVolumeClaim{},
		namespaces:     map[string]*corev1.Namespace{},
		nodes:          map[string]*corev1.Node{},
		events:         map[string][]eventsv1.Event{},
		nsEvents:       map[string][]eventsv1.Event{},
		netpols:        map[string][]networkingv1.NetworkPolicy{},
		logs:           map[string]string{},
		ForbiddenKinds: map[string]bool{},
	}
}

// DiscardLogger returns a logr.Logger that drops all output. Handy for tests.
func DiscardLogger() logr.Logger { return logr.Discard() }

// ----- population helpers -------------------------------------------------

func (f *FakeClient) AddPod(p *corev1.Pod)                                { f.pods[nsn(p.Namespace, p.Name)] = p }
func (f *FakeClient) AddDeployment(d *appsv1.Deployment)                  { f.deployments[nsn(d.Namespace, d.Name)] = d }
func (f *FakeClient) AddReplicaSet(rs *appsv1.ReplicaSet)                 { f.replicaSets[nsn(rs.Namespace, rs.Name)] = rs }
func (f *FakeClient) AddService(svc *corev1.Service)                      { f.services[nsn(svc.Namespace, svc.Name)] = svc }
func (f *FakeClient) AddEndpoints(ep *corev1.Endpoints)                   { f.endpoints[nsn(ep.Namespace, ep.Name)] = ep }
func (f *FakeClient) AddConfigMap(cm *corev1.ConfigMap)                   { f.configMaps[nsn(cm.Namespace, cm.Name)] = cm }
func (f *FakeClient) AddSecret(s *corev1.Secret)                          { f.secrets[nsn(s.Namespace, s.Name)] = s }
func (f *FakeClient) AddPVC(pvc *corev1.PersistentVolumeClaim)            { f.pvcs[nsn(pvc.Namespace, pvc.Name)] = pvc }
func (f *FakeClient) AddNamespace(ns *corev1.Namespace)                   { f.namespaces[ns.Name] = ns }
func (f *FakeClient) AddNode(n *corev1.Node)                              { f.nodes[n.Name] = n }
func (f *FakeClient) AddLogs(ns, pod, container, text string)             { f.logs[nsn(ns, pod)+"/"+container] = text }

// AddEventFor adds an event filed against a specific object.
func (f *FakeClient) AddEventFor(kind, ns, name string, ev eventsv1.Event) {
	k := kind + "|" + ns + "|" + name
	f.events[k] = append(f.events[k], ev)
	f.nsEvents[ns] = append(f.nsEvents[ns], ev)
}

// ----- Interface implementation -------------------------------------------

func (f *FakeClient) CurrentNamespace() string {
	if f.namespace == "" {
		return "default"
	}
	return f.namespace
}

func (f *FakeClient) GetPod(ctx context.Context, ns, name string) (*corev1.Pod, error) {
	if f.ForbiddenKinds["pods"] {
		return nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, name, fmt.Errorf("forbidden"))
	}
	p, ok := f.pods[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
	}
	return p, nil
}

func (f *FakeClient) GetDeployment(ctx context.Context, ns, name string) (*appsv1.Deployment, error) {
	if f.ForbiddenKinds["deployments"] {
		return nil, apierrors.NewForbidden(schema.GroupResource{Resource: "deployments"}, name, fmt.Errorf("forbidden"))
	}
	d, ok := f.deployments[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "deployments"}, name)
	}
	return d, nil
}

func (f *FakeClient) GetReplicaSet(ctx context.Context, ns, name string) (*appsv1.ReplicaSet, error) {
	rs, ok := f.replicaSets[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "replicasets"}, name)
	}
	return rs, nil
}

func (f *FakeClient) GetService(ctx context.Context, ns, name string) (*corev1.Service, error) {
	s, ok := f.services[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "services"}, name)
	}
	return s, nil
}

func (f *FakeClient) GetEndpoints(ctx context.Context, ns, name string) (*corev1.Endpoints, error) {
	ep, ok := f.endpoints[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "endpoints"}, name)
	}
	return ep, nil
}

func (f *FakeClient) GetConfigMap(ctx context.Context, ns, name string) (*corev1.ConfigMap, error) {
	cm, ok := f.configMaps[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, name)
	}
	return cm, nil
}

func (f *FakeClient) GetSecret(ctx context.Context, ns, name string) (*corev1.Secret, error) {
	s, ok := f.secrets[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name)
	}
	return s, nil
}

func (f *FakeClient) GetPVC(ctx context.Context, ns, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc, ok := f.pvcs[nsn(ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "persistentvolumeclaims"}, name)
	}
	return pvc, nil
}

func (f *FakeClient) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	ns, ok := f.namespaces[name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, name)
	}
	return ns, nil
}

func (f *FakeClient) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	n, ok := f.nodes[name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "nodes"}, name)
	}
	return n, nil
}

func (f *FakeClient) ListPods(ctx context.Context, ns string) ([]corev1.Pod, error) {
	var out []corev1.Pod
	for _, p := range f.pods {
		if ns == "" || p.Namespace == ns {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *FakeClient) ListDeployments(ctx context.Context, ns string) ([]appsv1.Deployment, error) {
	var out []appsv1.Deployment
	for _, d := range f.deployments {
		if ns == "" || d.Namespace == ns {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (f *FakeClient) ListReplicaSets(ctx context.Context, ns string) ([]appsv1.ReplicaSet, error) {
	var out []appsv1.ReplicaSet
	for _, rs := range f.replicaSets {
		if ns == "" || rs.Namespace == ns {
			out = append(out, *rs)
		}
	}
	return out, nil
}

func (f *FakeClient) ListServices(ctx context.Context, ns string) ([]corev1.Service, error) {
	var out []corev1.Service
	for _, s := range f.services {
		if ns == "" || s.Namespace == ns {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (f *FakeClient) ListNetworkPolicies(ctx context.Context, ns string) ([]networkingv1.NetworkPolicy, error) {
	return f.netpols[ns], nil
}

func (f *FakeClient) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	out := make([]corev1.Node, 0, len(f.nodes))
	for _, n := range f.nodes {
		out = append(out, *n)
	}
	return out, nil
}

func (f *FakeClient) ListEventsFor(ctx context.Context, kind, ns, name string) ([]eventsv1.Event, error) {
	return f.events[kind+"|"+ns+"|"+name], nil
}

func (f *FakeClient) ListEventsInNamespace(ctx context.Context, ns string) ([]eventsv1.Event, error) {
	if ns == "" {
		var all []eventsv1.Event
		for _, evs := range f.nsEvents {
			all = append(all, evs...)
		}
		return all, nil
	}
	return f.nsEvents[ns], nil
}

func (f *FakeClient) GetLogs(ctx context.Context, ns, pod, container string, _ int64, _ bool) (string, error) {
	return f.logs[nsn(ns, pod)+"/"+container], nil
}

func (f *FakeClient) CanI(ctx context.Context, verb, group, resource, ns string) (bool, error) {
	return true, nil
}

func nsn(ns, name string) string { return strings.Join([]string{ns, name}, "/") }
