package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/abialemuel/AI-Proxy-Service/config"
	userAPIhttp "github.com/abialemuel/AI-Proxy-Service/pkg/user/api/http"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	mainCfg "github.com/abialemuel/AI-Proxy-Service/config"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/cache"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/cache/redis"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/http/middleware/authguard"
	"github.com/abialemuel/AI-Proxy-Service/pkg/common/mongodb"
	oauthmanager "github.com/abialemuel/AI-Proxy-Service/pkg/common/oauth"
	userBusiness "github.com/abialemuel/AI-Proxy-Service/pkg/user/business"
	gpt4WebService "github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/gpt4_webservice"
	userRepository "github.com/abialemuel/AI-Proxy-Service/pkg/user/modules/repository"
	"github.com/abialemuel/poly-kit/infrastructure/apm"
	"github.com/abialemuel/poly-kit/infrastructure/logger"
	"github.com/labstack/echo/v4"
	mw "github.com/labstack/echo/v4/middleware"
	dd "gopkg.in/DataDog/dd-trace-go.v1/contrib/labstack/echo.v4"
)

var (
	APM *apm.APM
	log logger.Logger
)

func main() {
	// config
	cfg := initializeConfig("config.yaml")
	log = initializeLogger(cfg)

	// initialize apm
	if cfg.Get().APM.Enabled {
		host := fmt.Sprintf("%s:%d", cfg.Get().APM.Host, cfg.Get().APM.Port)
		apmPayload := apm.APMPayload{
			ServiceHost:    &host,
			ServiceName:    cfg.Get().App.Name,
			ServiceEnv:     cfg.Get().App.Env,
			ServiceTribe:   cfg.Get().App.Tribe,
			ServiceVersion: cfg.Get().App.Version,
			SampleRate:     cfg.Get().APM.Rate,
		}
		APM, err := apm.NewAPM(apm.DatadogAPMType, apmPayload)
		if err != nil {
			log.Get().Error(err)
			panic(err)
		}
		fmt.Println("APM started...")
		defer APM.EndAPM()
	}

	// init oauthProvider
	googleProvider, err := oauthmanager.NewOAuth2Provider(oauthmanager.GoogleProvider,
		"google",
		cfg.Get().GoogleOauth.ClientID,
		cfg.Get().GoogleOauth.ClientSecret,
		cfg.Get().GoogleOauth.RedirectURL)
	if err != nil {
		log.Get().Error(err)
		panic(err)
	}

	microsoftProvider, err := oauthmanager.NewOAuth2Provider(oauthmanager.MicrosoftProvider,
		cfg.Get().MicrosoftOauth.TenantID,
		cfg.Get().MicrosoftOauth.ClientID,
		cfg.Get().MicrosoftOauth.ClientSecret,
		cfg.Get().MicrosoftOauth.RedirectURL)
	if err != nil {
		log.Get().Error(err)
		panic(err)
	}

	// Init services
	url := fmt.Sprintf("%s:%d", cfg.Get().Redis.Host, cfg.Get().Redis.Port)
	redis.InitRedis(url, cfg.Get().Redis.Password, cfg.Get().Redis.DB)
	// Example usage of redis
	cache := cache.NewCache(&redis.Redis{})

	// init gpt4 webservice
	endpoint := fmt.Sprintf("%s%s", cfg.Get().OpenAI.Host, cfg.Get().OpenAI.Path)
	gpt4Webservice := gpt4WebService.NewGPT4WebService(endpoint, cfg.Get().OpenAI.ApiKey)

	// init mongoDB
	urlHost := fmt.Sprintf("%s:%d", cfg.Get().Mongo.Host, cfg.Get().Mongo.Port)
	mongoDB, err := mongodb.NewMongoDB(urlHost, cfg.Get().Mongo.Username, cfg.Get().Mongo.Password, cfg.Get().Mongo.DB)
	userRepo := userRepository.NewMongoDBRepository(mongoDB)

	// init userService
	userService := newUserService(cache, cfg.Get(), gpt4Webservice, userRepo)

	// Init HTTP client
	e := echo.New()
	e.Use()
	e.Use(mw.Recover())
	// opentelemetry echo middleware
	e.Use(otelecho.Middleware(cfg.Get().App.Name))
	// datadog echo middleware
	e.Use(dd.Middleware())
	e.Use(mw.CORSWithConfig(
		mw.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAuthorization},
			AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		}))

	//health check
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(200)
	})
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// run server
	go func() {
		address := fmt.Sprintf(":%d", cfg.Get().App.Port)

		if err := e.Start(address); err != nil {
			log.Get().Info("shutting down the server")
		}
	}()

	authGuard := authguard.NewAuthGuard(*cfg.Get())
	authGuard.AddService(cfg.Get().Services)

	// Register API
	userHandler := userAPIhttp.NewHandler(userService, googleProvider, microsoftProvider, cfg.Get())
	userAPIhttp.RegisterPath(e, userHandler, authGuard)

	// Wait for interrupt signal to gracefully shutdown the server with
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// a timeout of 10 seconds to shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Get().Fatal(err)
	}
}

func initializeLogger(cfg mainCfg.Config) logger.Logger {
	fmt.Printf("%s started...\n", cfg.Get().App.Name)
	log := logger.New().Init(logger.Config{
		Level:  cfg.Get().Log.Level,
		Format: cfg.Get().Log.Format,
	})
	return log
}

func initializeConfig(path string) mainCfg.Config {
	cfg := mainCfg.New()
	err := cfg.Init(path)
	if err != nil {
		fmt.Errorf("failed to load config: %s", err.Error())
		panic(err)
	}
	return cfg
}

func newUserService(
	cache *cache.Cache,
	cfg *config.MainConfig,
	gpt4Webservice gpt4WebService.GPT4WebService,
	userRepo *userRepository.MongoDBRepository,
) userBusiness.UserService {
	userService := userBusiness.NewUserService(userRepo, cache, cfg, gpt4Webservice)
	return userService
}
