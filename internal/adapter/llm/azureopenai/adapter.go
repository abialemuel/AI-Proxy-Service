// Package azureopenai adapts Azure OpenAI's deployment-based endpoint to the
// LLMProvider port. It reuses the OpenAI-compatible wire types but uses the
// "api-key" header and deployment-scoped URL form.
package azureopenai

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

type Config struct {
	Name       string
	BaseURL    string // e.g. https://myaoai.openai.azure.com
	APIKey     string
	APIVersion string // e.g. 2024-02-15-preview
	Deployment string // Azure deployment name
	Timeout    time.Duration
	Models     []string
	HTTP       *http.Client
}

type Adapter struct {
	cfg    Config
	client *http.Client
	models map[string]struct{}
}

func New(cfg Config) *Adapter {
	if cfg.Name == "" {
		cfg.Name = "azure-openai"
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2024-02-15-preview"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	models := map[string]struct{}{}
	for _, m := range cfg.Models {
		models[strings.ToLower(m)] = struct{}{}
	}
	return &Adapter{cfg: cfg, client: hc, models: models}
}

func (a *Adapter) Name() string { return a.cfg.Name }

func (a *Adapter) Supports(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	if _, ok := a.models[m]; ok {
		return true
	}
	// Azure routes by deployment, so when allowlist is empty we accept any
	// model id and rely on Deployment to resolve.
	return len(a.models) == 0
}

func (a *Adapter) endpoint() string {
	return fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		a.cfg.BaseURL, a.cfg.Deployment, a.cfg.APIVersion)
}

func (a *Adapter) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	body := toOpenAIRequest(req)
	body.Stream = false

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "marshal request", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint(), bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", a.cfg.APIKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderTimeout, "azure timeout", err)
		}
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "azure call", err)
	}
	defer resp.Body.Close()

	rawResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return domain.ChatResponse{}, apierrors.New(apierrors.CodeRateLimited,
				"Azure OpenAI rate limit exceeded; please retry later")
		}
		return domain.ChatResponse{}, apierrors.New(apierrors.CodeProviderError,
			fmt.Sprintf("azure returned %d: %s", resp.StatusCode, truncate(string(rawResp), 512)))
	}

	var parsed openaicompat.ChatResponse
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderBadReply, "decode azure response", err)
	}
	return fromOpenAIResponse(a.cfg.Name, parsed), nil
}

func (a *Adapter) Stream(_ context.Context, _ domain.ChatRequest) (<-chan domain.StreamChunk, error) {
	return nil, ports.ErrStreamingUnsupported
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
