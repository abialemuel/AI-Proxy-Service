package ports

import (
	"context"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
)

// ProviderRegistry resolves a domain.ChatRequest to a concrete LLMProvider.
// This is the central Strategy + Registry component of the proxy.
type ProviderRegistry interface {
	Register(p LLMProvider)
	Resolve(req domain.ChatRequest) (LLMProvider, error)
	List() []string
}

// ErrStreamingUnsupported should be returned by LLMProvider.Stream when the
// provider has no streaming capability.
var ErrStreamingUnsupported = apierrors.New(apierrors.CodeProviderError, "streaming not supported by provider")

// Compile-time guard: keep context import alive even if unused in subtypes.
var _ = context.Background
