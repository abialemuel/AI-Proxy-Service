// Package openai is the LLMProvider adapter for the native OpenAI Chat
// Completions API. It also serves OpenAI-compatible gateways (e.g. LM Studio,
// vLLM, OpenRouter) when given the appropriate baseURL.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
	"github.com/abialemuel/AI-Proxy-Service/pkg/openaicompat"
)

// Config configures the OpenAI adapter.
type Config struct {
	Name    string // optional override (defaults to "openai")
	BaseURL string // e.g. https://api.openai.com/v1
	APIKey  string
	Timeout time.Duration
	// Models declares which model ids this instance serves. Empty = wildcard
	// for the "gpt-" prefix and "o1"/"o3" family.
	Models []string
	HTTP   *http.Client
}

// Adapter implements ports.LLMProvider against the OpenAI Chat Completions API.
type Adapter struct {
	cfg    Config
	client *http.Client
	models map[string]struct{}
}

// New constructs a new OpenAI adapter.
func New(cfg Config) *Adapter {
	if cfg.Name == "" {
		cfg.Name = "openai"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 20,
			},
		}
	}
	models := make(map[string]struct{}, len(cfg.Models))
	for _, m := range cfg.Models {
		models[strings.ToLower(m)] = struct{}{}
	}
	return &Adapter{cfg: cfg, client: hc, models: models}
}

// Name returns the provider identifier.
func (a *Adapter) Name() string { return a.cfg.Name }

// Supports returns whether this adapter claims the given model id.
func (a *Adapter) Supports(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	if _, ok := a.models[m]; ok {
		return true
	}
	if len(a.models) > 0 {
		return false
	}
	// Heuristic fallback for common OpenAI model families.
	switch {
	case strings.HasPrefix(m, "gpt-"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "chatgpt-"):
		return true
	}
	return false
}

// Chat performs a non-streaming completion.
func (a *Adapter) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	body := toOpenAIRequest(req)
	body.Stream = false

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "marshal request", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.cfg.APIKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderTimeout, "openai timeout", err)
		}
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "openai call", err)
	}
	defer resp.Body.Close()

	rawResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return domain.ChatResponse{}, mapHTTPError(resp.StatusCode, rawResp)
	}

	var parsed openaicompat.ChatResponse
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderBadReply, "decode openai response", err)
	}
	return fromOpenAIResponse(a.cfg.Name, parsed), nil
}

// Stream is a stub for now; SSE plumbing lives in adapter.go-stream (future).
func (a *Adapter) Stream(ctx context.Context, req domain.ChatRequest) (<-chan domain.StreamChunk, error) {
	return nil, ports.ErrStreamingUnsupported
}

func mapHTTPError(status int, body []byte) error {
	switch status {
	case http.StatusTooManyRequests:
		return apierrors.New(apierrors.CodeRateLimited, "provider rate limited")
	case http.StatusUnauthorized, http.StatusForbidden:
		return apierrors.New(apierrors.CodeUnauthorized, "provider rejected credentials")
	}
	return apierrors.New(apierrors.CodeProviderError,
		fmt.Sprintf("provider returned %d: %s", status, truncate(string(body), 512)))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
