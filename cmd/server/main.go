// Command server is the AI Proxy Service entry point.
//
// Wiring is intentionally explicit: configuration → adapters → use cases →
// transports. No global state, no init() side-effects.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cacheredis "github.com/abialemuel/AI-Proxy-Service/internal/adapter/cache/redis"
	httpsrv "github.com/abialemuel/AI-Proxy-Service/internal/adapter/http"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/http/handler"
	llm "github.com/abialemuel/AI-Proxy-Service/internal/adapter/llm"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/llm/anthropic"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/llm/azureopenai"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/llm/gemini"
	"github.com/abialemuel/AI-Proxy-Service/internal/adapter/llm/openai"
	mongoadp "github.com/abialemuel/AI-Proxy-Service/internal/adapter/persistence/mongo"
	"github.com/abialemuel/AI-Proxy-Service/internal/pkg/config"
	"github.com/abialemuel/AI-Proxy-Service/internal/ports"
	"github.com/abialemuel/AI-Proxy-Service/internal/usecase"
)

func main() {
	cfgPath := envOr("CONFIG_PATH", "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("%s starting (env=%s, version=%s)", cfg.App.Name, cfg.App.Env, cfg.App.Version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Driven adapters ---
	cache := cacheredis.New(
		fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		cfg.Redis.Password, cfg.Redis.DB,
	)
	if err := cache.Ping(ctx); err != nil {
		log.Printf("warning: redis unreachable: %v (continuing without quotas)", err)
	}

	db, err := mongoadp.Connect(ctx, cfg.Mongo.Host, cfg.Mongo.Port, cfg.Mongo.Username, cfg.Mongo.Password, cfg.Mongo.DB)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	convRepo := mongoadp.NewConversationRepo(db)

	// --- LLM provider registry (Adapter + Strategy + Registry pattern) ---
	registry := llm.NewRegistry(envOr("DEFAULT_PROVIDER", ""))
	registerProviders(registry, cfg)

	if names := registry.List(); len(names) == 0 {
		log.Fatalf("no LLM providers enabled in config.providers — at least one is required")
	} else {
		log.Printf("LLM providers registered: %s", strings.Join(names, ", "))
	}

	// --- Use cases ---
	quota := usecase.NewQuotaService(cache, cfg.Quota.WindowSec, cfg.Quota.TokensPerWindow)
	chatSvc := usecase.NewChatService(registry, convRepo, quota)

	// --- Driving adapters ---
	chatHandler := handler.NewChatHandler(chatSvc)
	server := httpsrv.New(cfg, chatHandler)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	go func() {
		log.Printf("listening on %s", addr)
		if err := server.E.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.E.Shutdown(shCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
	_ = cache.Close()
}

// registerProviders applies the providers map from config. Each entry is
// gated by .enabled, so operators can mix and match.
func registerProviders(reg ports.ProviderRegistry, cfg *config.Config) {
	for name, p := range cfg.Providers {
		if !p.Enabled {
			continue
		}
		timeout := time.Duration(p.TimeoutSec) * time.Second
		switch strings.ToLower(name) {
		case "openai":
			reg.Register(openai.New(openai.Config{
				Name: name, BaseURL: p.BaseURL, APIKey: p.APIKey,
				Timeout: timeout, Models: p.Models,
			}))
		case "azure-openai", "azureopenai", "azure":
			reg.Register(azureopenai.New(azureopenai.Config{
				Name: name, BaseURL: p.BaseURL, APIKey: p.APIKey,
				APIVersion: p.APIVersion, Deployment: p.Deployment,
				Timeout: timeout, Models: p.Models,
			}))
		case "anthropic", "claude":
			reg.Register(anthropic.New(anthropic.Config{
				Name: name, BaseURL: p.BaseURL, APIKey: p.APIKey,
				APIVersion: p.APIVersion, Timeout: timeout, Models: p.Models,
			}))
		case "gemini", "google":
			reg.Register(gemini.New(gemini.Config{
				Name: name, BaseURL: p.BaseURL, APIKey: p.APIKey,
				Timeout: timeout, Models: p.Models,
			}))
		default:
			log.Printf("unknown provider in config: %s (skipped)", name)
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
