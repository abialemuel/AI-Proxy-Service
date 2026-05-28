// Package anthropic adapts Anthropic's Messages API
// (https://docs.anthropic.com/en/api/messages) to the LLMProvider port.
package anthropic

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
)

const defaultVersion = "2023-06-01"

type Config struct {
	Name       string
	BaseURL    string // default https://api.anthropic.com
	APIKey     string
	APIVersion string
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
		cfg.Name = "anthropic"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.APIVersion == "" {
		cfg.APIVersion = defaultVersion
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 90 * time.Second
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
	if len(a.models) > 0 {
		return false
	}
	return strings.HasPrefix(m, "claude-")
}

// --- vendor wire types ---

type wireRequest struct {
	Model       string        `json:"model"`
	System      string        `json:"system,omitempty"`
	Messages    []wireMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type wireMessage struct {
	Role    string      `json:"role"`
	Content []wirePart  `json:"content"`
}

type wirePart struct {
	Type   string      `json:"type"`
	Text   string      `json:"text,omitempty"`
	Source *wireSource `json:"source,omitempty"`
}

type wireSource struct {
	Type      string `json:"type"`       // "url" or "base64"
	URL       string `json:"url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type wireResponse struct {
	ID         string     `json:"id"`
	Model      string     `json:"model"`
	Role       string     `json:"role"`
	Content    []wirePart `json:"content"`
	StopReason string     `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// --- methods ---

func (a *Adapter) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	body := toWire(req)

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", a.cfg.APIVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderTimeout, "anthropic timeout", err)
		}
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "anthropic call", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return domain.ChatResponse{}, apierrors.New(apierrors.CodeRateLimited, "anthropic rate limited")
		case http.StatusUnauthorized, http.StatusForbidden:
			return domain.ChatResponse{}, apierrors.New(apierrors.CodeUnauthorized, "anthropic auth failed")
		}
		return domain.ChatResponse{}, apierrors.New(apierrors.CodeProviderError,
			fmt.Sprintf("anthropic returned %d: %s", resp.StatusCode, truncate(string(respBody), 512)))
	}

	var parsed wireResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderBadReply, "decode anthropic response", err)
	}
	return fromWire(a.cfg.Name, parsed), nil
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
