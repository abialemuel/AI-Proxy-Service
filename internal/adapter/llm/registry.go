// Package llm wires concrete provider adapters into a runtime registry.
package llm

import (
	"strings"
	"sync"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

// Registry is the default in-memory ports.ProviderRegistry.
type Registry struct {
	mu        sync.RWMutex
	providers []ports.LLMProvider
	byName    map[string]ports.LLMProvider
	// default provider used when no explicit hint matches and no provider claims the model.
	defaultName string
}

// NewRegistry constructs an empty registry. defaultName may be empty.
func NewRegistry(defaultName string) *Registry {
	return &Registry{
		byName:      map[string]ports.LLMProvider{},
		defaultName: strings.ToLower(strings.TrimSpace(defaultName)),
	}
}

// Register adds a provider; later registrations override earlier ones by name.
func (r *Registry) Register(p ports.LLMProvider) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	name := strings.ToLower(p.Name())
	if _, dup := r.byName[name]; !dup {
		r.providers = append(r.providers, p)
	} else {
		// replace in slice
		for i, existing := range r.providers {
			if strings.ToLower(existing.Name()) == name {
				r.providers[i] = p
				break
			}
		}
	}
	r.byName[name] = p
}

// Resolve picks a provider for the request using, in order:
//  1. explicit Provider hint
//  2. first provider whose Supports(model) returns true
//  3. configured default
func (r *Registry) Resolve(req domain.ChatRequest) (ports.LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if hint := strings.ToLower(strings.TrimSpace(req.Provider)); hint != "" {
		if p, ok := r.byName[hint]; ok {
			return p, nil
		}
		return nil, apierrors.New(apierrors.CodeUnsupportedModel,
			"unknown provider hint: "+req.Provider)
	}

	for _, p := range r.providers {
		if p.Supports(req.Model) {
			return p, nil
		}
	}

	if r.defaultName != "" {
		if p, ok := r.byName[r.defaultName]; ok {
			return p, nil
		}
	}
	return nil, apierrors.New(apierrors.CodeUnsupportedModel,
		"no provider can serve model: "+req.Model)
}

// List returns the registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p.Name())
	}
	return out
}
