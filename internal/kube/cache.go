package kube

import (
	"context"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/go-logr/logr"
)

// ResourceCache is a request-scoped memoization layer in front of an
// Interface. It dedupes fetches within one engine.Run so that rules touching
// the same resource share the API call.
//
// Cache semantics:
//   - First call: fetches; records either the object or the error.
//   - Subsequent calls: returns the cached result (including cached errors).
//   - Negative entries (NotFound, Forbidden) are cached to avoid re-asking
//     the API server for objects we know don't exist or aren't visible.
//
// Safe for concurrent use, though engine.Run is single-goroutine today.
type ResourceCache struct {
	client Interface
	logger logr.Logger
	mu     sync.Mutex

	pods        map[string]podEntry
	deployments map[string]deployEntry
	replicaSets map[string]rsEntry
	services    map[string]svcEntry
	endpoints   map[string]epEntry
	configMaps  map[string]cmEntry
	secrets     map[string]secretEntry
	pvcs        map[string]pvcEntry
	namespaces  map[string]nsEntry
	nodes       map[string]nodeEntry

	podLists        map[string]listEntry[corev1.Pod]
	deployLists     map[string]listEntry[appsv1.Deployment]
	rsLists         map[string]listEntry[appsv1.ReplicaSet]
	svcLists        map[string]listEntry[corev1.Service]
	netpolLists     map[string]listEntry[networkingv1.NetworkPolicy]
	quotaLists      map[string]listEntry[corev1.ResourceQuota]
	nodeList        *listEntry[corev1.Node]
	eventFor        map[string]listEntry[eventsv1.Event] // key: kind|ns|name
	eventInNS       map[string]listEntry[eventsv1.Event] // key: namespace ("" = all)
}

type listEntry[T any] struct {
	items []T
	err   error
}

type podEntry struct {
	obj *corev1.Pod
	err error
}
type deployEntry struct {
	obj *appsv1.Deployment
	err error
}
type rsEntry struct {
	obj *appsv1.ReplicaSet
	err error
}
type svcEntry struct {
	obj *corev1.Service
	err error
}
type epEntry struct {
	obj *corev1.Endpoints
	err error
}
type cmEntry struct {
	obj *corev1.ConfigMap
	err error
}
type secretEntry struct {
	obj *corev1.Secret
	err error
}
type pvcEntry struct {
	obj *corev1.PersistentVolumeClaim
	err error
}
type nsEntry struct {
	obj *corev1.Namespace
	err error
}
type nodeEntry struct {
	obj *corev1.Node
	err error
}

// NewResourceCache creates a cache bound to the given client.
func NewResourceCache(client Interface, logger logr.Logger) *ResourceCache {
	if logger.GetSink() == nil {
		logger = logr.Discard()
	}
	return &ResourceCache{
		client:      client,
		logger:      logger,
		pods:        map[string]podEntry{},
		deployments: map[string]deployEntry{},
		replicaSets: map[string]rsEntry{},
		services:    map[string]svcEntry{},
		endpoints:   map[string]epEntry{},
		configMaps:  map[string]cmEntry{},
		secrets:     map[string]secretEntry{},
		pvcs:        map[string]pvcEntry{},
		namespaces:  map[string]nsEntry{},
		nodes:       map[string]nodeEntry{},
		podLists:    map[string]listEntry[corev1.Pod]{},
		deployLists: map[string]listEntry[appsv1.Deployment]{},
		rsLists:     map[string]listEntry[appsv1.ReplicaSet]{},
		svcLists:    map[string]listEntry[corev1.Service]{},
		netpolLists: map[string]listEntry[networkingv1.NetworkPolicy]{},
		quotaLists:  map[string]listEntry[corev1.ResourceQuota]{},
		eventFor:    map[string]listEntry[eventsv1.Event]{},
		eventInNS:   map[string]listEntry[eventsv1.Event]{},
	}
}

// Client returns the underlying client. Rules should prefer cache methods
// and only fall back to the raw client for rare cases.
func (c *ResourceCache) Client() Interface { return c.client }

func key(ns, name string) string { return ns + "/" + name }

// -----------------------------------------------------------------------------
// Typed getters
// -----------------------------------------------------------------------------

// GetPod returns (pod, found, err). found==false with err==nil means
// "not found in cluster". A forbidden response returns err==ErrForbidden.
func (c *ResourceCache) GetPod(ctx context.Context, ns, name string) (*corev1.Pod, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.pods[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetPod(ctx, ns, name)
	return c.storePod(k, obj, err)
}

func (c *ResourceCache) storePod(k string, obj *corev1.Pod, err error) (*corev1.Pod, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.pods[k] = podEntry{obj: nil, err: nil}
		return nil, false, nil
	case IsForbidden(err):
		c.pods[k] = podEntry{obj: nil, err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.pods[k] = podEntry{obj: obj, err: nil}
	return obj, true, nil
}

func (c *ResourceCache) GetDeployment(ctx context.Context, ns, name string) (*appsv1.Deployment, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.deployments[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetDeployment(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.deployments[k] = deployEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.deployments[k] = deployEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.deployments[k] = deployEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetReplicaSet(ctx context.Context, ns, name string) (*appsv1.ReplicaSet, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.replicaSets[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetReplicaSet(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.replicaSets[k] = rsEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.replicaSets[k] = rsEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.replicaSets[k] = rsEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetService(ctx context.Context, ns, name string) (*corev1.Service, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.services[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetService(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.services[k] = svcEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.services[k] = svcEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.services[k] = svcEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetEndpoints(ctx context.Context, ns, name string) (*corev1.Endpoints, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.endpoints[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetEndpoints(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.endpoints[k] = epEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.endpoints[k] = epEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.endpoints[k] = epEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetConfigMap(ctx context.Context, ns, name string) (*corev1.ConfigMap, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.configMaps[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetConfigMap(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.configMaps[k] = cmEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.configMaps[k] = cmEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.configMaps[k] = cmEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetSecret(ctx context.Context, ns, name string) (*corev1.Secret, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.secrets[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetSecret(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.secrets[k] = secretEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.secrets[k] = secretEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.secrets[k] = secretEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetPVC(ctx context.Context, ns, name string) (*corev1.PersistentVolumeClaim, bool, error) {
	k := key(ns, name)
	c.mu.Lock()
	if e, ok := c.pvcs[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetPVC(ctx, ns, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.pvcs[k] = pvcEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.pvcs[k] = pvcEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.pvcs[k] = pvcEntry{obj: obj}
	return obj, true, nil
}

func (c *ResourceCache) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, bool, error) {
	k := key("", name)
	c.mu.Lock()
	if e, ok := c.namespaces[k]; ok {
		c.mu.Unlock()
		return e.obj, e.obj != nil, e.err
	}
	c.mu.Unlock()
	obj, err := c.client.GetNamespace(ctx, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case IsNotFound(err):
		c.namespaces[k] = nsEntry{}
		return nil, false, nil
	case IsForbidden(err):
		c.namespaces[k] = nsEntry{err: ErrForbidden}
		return nil, false, ErrForbidden
	case err != nil:
		return nil, false, err
	}
	c.namespaces[k] = nsEntry{obj: obj}
	return obj, true, nil
}

// -----------------------------------------------------------------------------
// Typed listers
// -----------------------------------------------------------------------------

func (c *ResourceCache) ListPods(ctx context.Context, ns string) ([]corev1.Pod, error) {
	c.mu.Lock()
	if e, ok := c.podLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListPods(ctx, ns)
	c.mu.Lock()
	c.podLists[ns] = listEntry[corev1.Pod]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListDeployments(ctx context.Context, ns string) ([]appsv1.Deployment, error) {
	c.mu.Lock()
	if e, ok := c.deployLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListDeployments(ctx, ns)
	c.mu.Lock()
	c.deployLists[ns] = listEntry[appsv1.Deployment]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListReplicaSets(ctx context.Context, ns string) ([]appsv1.ReplicaSet, error) {
	c.mu.Lock()
	if e, ok := c.rsLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListReplicaSets(ctx, ns)
	c.mu.Lock()
	c.rsLists[ns] = listEntry[appsv1.ReplicaSet]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListServices(ctx context.Context, ns string) ([]corev1.Service, error) {
	c.mu.Lock()
	if e, ok := c.svcLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListServices(ctx, ns)
	c.mu.Lock()
	c.svcLists[ns] = listEntry[corev1.Service]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListNetworkPolicies(ctx context.Context, ns string) ([]networkingv1.NetworkPolicy, error) {
	c.mu.Lock()
	if e, ok := c.netpolLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListNetworkPolicies(ctx, ns)
	c.mu.Lock()
	c.netpolLists[ns] = listEntry[networkingv1.NetworkPolicy]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListResourceQuotas(ctx context.Context, ns string) ([]corev1.ResourceQuota, error) {
	c.mu.Lock()
	if e, ok := c.quotaLists[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListResourceQuotas(ctx, ns)
	c.mu.Lock()
	c.quotaLists[ns] = listEntry[corev1.ResourceQuota]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

func (c *ResourceCache) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	c.mu.Lock()
	if c.nodeList != nil {
		e := *c.nodeList
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListNodes(ctx)
	c.mu.Lock()
	c.nodeList = &listEntry[corev1.Node]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

// ListEventsFor fetches Warning+Normal events referring to the given object.
func (c *ResourceCache) ListEventsFor(ctx context.Context, kind, ns, name string) ([]eventsv1.Event, error) {
	k := kind + "|" + ns + "|" + name
	c.mu.Lock()
	if e, ok := c.eventFor[k]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListEventsFor(ctx, kind, ns, name)
	c.mu.Lock()
	c.eventFor[k] = listEntry[eventsv1.Event]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

// ListEventsInNamespace returns all events in a namespace ("" = all namespaces).
func (c *ResourceCache) ListEventsInNamespace(ctx context.Context, ns string) ([]eventsv1.Event, error) {
	c.mu.Lock()
	if e, ok := c.eventInNS[ns]; ok {
		c.mu.Unlock()
		return e.items, e.err
	}
	c.mu.Unlock()
	items, err := c.client.ListEventsInNamespace(ctx, ns)
	c.mu.Lock()
	c.eventInNS[ns] = listEntry[eventsv1.Event]{items: items, err: err}
	c.mu.Unlock()
	return items, err
}

// Logs is a thin forward to the client. Not cached (rules request logs only
// once per container per run).
func (c *ResourceCache) Logs(ctx context.Context, ns, pod, container string, tailLines int64, previous bool) (string, error) {
	return c.client.GetLogs(ctx, ns, pod, container, tailLines, previous)
}

// CanI is a thin forward to the client.
func (c *ResourceCache) CanI(ctx context.Context, verb, group, resource, ns string) (bool, error) {
	return c.client.CanI(ctx, verb, group, resource, ns)
}
