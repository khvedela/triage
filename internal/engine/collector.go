package engine

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/kube"
)

// Collector prefetches related resources for a target into the shared cache.
// Prefetching is advisory: the cache is lazy and rules will fault in what
// they need even if prefetch is skipped. Prefetch exists to batch HTTP/2
// streams and warm the cache for all rules, reducing p99 latency.
type Collector struct {
	Client kube.Interface
	Cache  *kube.ResourceCache
	Logger logr.Logger
}

// Prefetch fills the cache with the target and its commonly-referenced
// related objects. Errors are joined and returned but never fatal for the run.
func (c *Collector) Prefetch(ctx context.Context, t findings.Target, includeEvents, includeRelated bool) error {
	switch t.Kind {
	case findings.TargetKindPod:
		return c.prefetchPod(ctx, t.Namespace, t.Name, includeEvents, includeRelated)
	case findings.TargetKindDeployment:
		return c.prefetchDeployment(ctx, t.Namespace, t.Name, includeEvents, includeRelated)
	case findings.TargetKindNamespace:
		return c.prefetchNamespace(ctx, t.Name, includeEvents)
	case findings.TargetKindCluster:
		return c.prefetchCluster(ctx, includeEvents)
	}
	return fmt.Errorf("collector: unknown target kind %q", t.Kind)
}

func (c *Collector) prefetchPod(ctx context.Context, ns, name string, events, related bool) error {
	// Pod itself.
	if _, _, err := c.Cache.GetPod(ctx, ns, name); err != nil {
		return err
	}
	if events {
		if _, err := c.Cache.ListEventsFor(ctx, "Pod", ns, name); err != nil {
			c.Logger.V(1).Info("events prefetch failed", "err", err)
		}
	}
	if related {
		// Services / endpoints that select this pod require label matching;
		// defer to rule-level lazy fetch via the cache.
		_, _ = c.Cache.ListServices(ctx, ns)
	}
	return nil
}

func (c *Collector) prefetchDeployment(ctx context.Context, ns, name string, events, related bool) error {
	if _, _, err := c.Cache.GetDeployment(ctx, ns, name); err != nil {
		return err
	}
	if events {
		_, _ = c.Cache.ListEventsFor(ctx, "Deployment", ns, name)
	}
	if related {
		_, _ = c.Cache.ListReplicaSets(ctx, ns)
		_, _ = c.Cache.ListPods(ctx, ns)
	}
	return nil
}

func (c *Collector) prefetchNamespace(ctx context.Context, ns string, events bool) error {
	if _, _, err := c.Cache.GetNamespace(ctx, ns); err != nil {
		return err
	}
	_, _ = c.Cache.ListPods(ctx, ns)
	_, _ = c.Cache.ListServices(ctx, ns)
	_, _ = c.Cache.ListDeployments(ctx, ns)
	if events {
		_, _ = c.Cache.ListEventsInNamespace(ctx, ns)
	}
	return nil
}

func (c *Collector) prefetchCluster(ctx context.Context, events bool) error {
	_, _ = c.Cache.ListNodes(ctx)
	if events {
		_, _ = c.Cache.ListEventsInNamespace(ctx, "") // all namespaces
	}
	return nil
}
