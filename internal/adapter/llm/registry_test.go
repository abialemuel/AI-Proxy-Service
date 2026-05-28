package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
)

type fakeProvider struct {
	name    string
	prefix  string
	chatCnt int
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Supports(model string) bool {
	return strings.HasPrefix(strings.ToLower(model), f.prefix)
}
func (f *fakeProvider) Chat(_ context.Context, _ domain.ChatRequest) (domain.ChatResponse, error) {
	f.chatCnt++
	return domain.ChatResponse{Provider: f.name}, nil
}
func (f *fakeProvider) Stream(_ context.Context, _ domain.ChatRequest) (<-chan domain.StreamChunk, error) {
	return nil, nil
}

func TestRegistry_ResolveByModelPrefix(t *testing.T) {
	r := NewRegistry("")
	r.Register(&fakeProvider{name: "openai", prefix: "gpt-"})
	r.Register(&fakeProvider{name: "anthropic", prefix: "claude-"})

	p, err := r.Resolve(domain.ChatRequest{Model: "claude-3-7-sonnet"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Fatalf("want anthropic, got %s", p.Name())
	}
}

func TestRegistry_ProviderHintOverridesPrefix(t *testing.T) {
	r := NewRegistry("")
	r.Register(&fakeProvider{name: "openai", prefix: "gpt-"})
	r.Register(&fakeProvider{name: "anthropic", prefix: "claude-"})

	p, err := r.Resolve(domain.ChatRequest{Model: "gpt-4o", Provider: "anthropic"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Name() != "anthropic" {
		t.Fatalf("hint should win; got %s", p.Name())
	}
}

func TestRegistry_FallbackToDefault(t *testing.T) {
	r := NewRegistry("openai")
	r.Register(&fakeProvider{name: "openai", prefix: "gpt-"})
	r.Register(&fakeProvider{name: "anthropic", prefix: "claude-"})

	p, err := r.Resolve(domain.ChatRequest{Model: "custom-model"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Name() != "openai" {
		t.Fatalf("want default openai, got %s", p.Name())
	}
}

func TestRegistry_UnknownProviderHint(t *testing.T) {
	r := NewRegistry("")
	r.Register(&fakeProvider{name: "openai", prefix: "gpt-"})

	if _, err := r.Resolve(domain.ChatRequest{Model: "gpt-4o", Provider: "nope"}); err == nil {
		t.Fatal("expected error for unknown provider hint")
	}
}
