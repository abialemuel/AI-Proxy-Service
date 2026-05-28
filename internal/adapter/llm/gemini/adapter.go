// Package gemini adapts the Google Generative Language API
// (https://ai.google.dev/api/rest/v1/models/generateContent) to LLMProvider.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/internal/domain"
	apierrors "github.com/abialemuel/AI-Proxy-Service/internal/pkg/errors"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
)

type Config struct {
	Name    string
	BaseURL string // default https://generativelanguage.googleapis.com
	APIKey  string
	Timeout time.Duration
	Models  []string
	HTTP    *http.Client
}

type Adapter struct {
	cfg    Config
	client *http.Client
	models map[string]struct{}
}

func New(cfg Config) *Adapter {
	if cfg.Name == "" {
		cfg.Name = "gemini"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com"
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
	if len(a.models) > 0 {
		return false
	}
	return strings.HasPrefix(m, "gemini-") || strings.HasPrefix(m, "models/gemini-")
}

// vendor wire types

type wirePart struct {
	Text string `json:"text,omitempty"`
}

type wireContent struct {
	Role  string     `json:"role,omitempty"` // "user" or "model"
	Parts []wirePart `json:"parts"`
}

type wireGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type wireRequest struct {
	SystemInstruction *wireContent          `json:"systemInstruction,omitempty"`
	Contents          []wireContent         `json:"contents"`
	GenerationConfig  *wireGenerationConfig `json:"generationConfig,omitempty"`
}

type wireResponse struct {
	Candidates []struct {
		Content      wireContent `json:"content"`
		FinishReason string      `json:"finishReason"`
		Index        int         `json:"index"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (a *Adapter) endpoint(model string) string {
	id := strings.TrimPrefix(model, "models/")
	return fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		a.cfg.BaseURL, url.PathEscape(id), url.QueryEscape(a.cfg.APIKey))
}

func (a *Adapter) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	body := toWire(req)

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "marshal request", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint(req.Model), bytes.NewReader(raw))
	if err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderTimeout, "gemini timeout", err)
		}
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderError, "gemini call", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return domain.ChatResponse{}, apierrors.New(apierrors.CodeRateLimited, "gemini rate limited")
		case http.StatusUnauthorized, http.StatusForbidden:
			return domain.ChatResponse{}, apierrors.New(apierrors.CodeUnauthorized, "gemini auth failed")
		}
		return domain.ChatResponse{}, apierrors.New(apierrors.CodeProviderError,
			fmt.Sprintf("gemini returned %d: %s", resp.StatusCode, truncate(string(respBody), 512)))
	}

	var parsed wireResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return domain.ChatResponse{}, apierrors.Wrap(apierrors.CodeProviderBadReply, "decode gemini response", err)
	}
	return fromWire(a.cfg.Name, req.Model, parsed), nil
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
