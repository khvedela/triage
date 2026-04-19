// Package rules holds the built-in rule set and the registry that lets
// each rule self-register in its init(). The engine discovers rules by
// calling All(); no explicit wiring is required.
package rules

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"

	"github.com/khvedela/kubediag/internal/findings"
	"github.com/khvedela/kubediag/internal/kube"
)

// Rule is the interface every diagnosis rule implements.
//
// Rules are stateless: the engine calls Evaluate once per target and then
// discards the rule. All per-run state must live on Context, not on the rule.
type Rule interface {
	Meta() findings.RuleMeta
	Evaluate(ctx context.Context, rc *Context) ([]findings.Finding, error)
}

// Context is the per-run handle a rule gets. It wraps the resource cache
// so rules never touch client-go directly — this keeps them pure, testable,
// and trivial to stub.
type Context struct {
	Target findings.Target
	Cache  *kube.ResourceCache
	Logger logr.Logger
	Now    func() time.Time
}

// Clock returns a safe time.Now function.
func (c *Context) Clock() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Rule{}
)

// Register adds a rule to the global registry. Intended for call from init().
// Panics on duplicate ID because that always indicates a programming error.
func Register(r Rule) {
	meta := r.Meta()
	if meta.ID == "" {
		panic("rules.Register: empty rule ID")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[meta.ID]; exists {
		panic("rules.Register: duplicate rule ID " + meta.ID)
	}
	registry[meta.ID] = r
}

// All returns every registered rule, sorted by ID for deterministic iteration.
func All() []Rule {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta().ID < out[j].Meta().ID })
	return out
}

// Get returns the rule with the given ID, or nil if unregistered.
func Get(id string) Rule {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[id]
}
