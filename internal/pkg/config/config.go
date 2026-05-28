// Package config loads typed configuration from a YAML file with env overrides.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the strongly-typed top-level configuration.
type Config struct {
	App       App                 `mapstructure:"app"`
	Log       Log                 `mapstructure:"log"`
	HTTP      HTTP                `mapstructure:"http"`
	APM       APM                 `mapstructure:"apm"`
	Redis     Redis               `mapstructure:"redis"`
	Mongo     Mongo               `mapstructure:"mongo"`
	OAuth     OAuth               `mapstructure:"oauth"`
	Providers map[string]Provider `mapstructure:"providers"`
	Services  []ServiceCreds      `mapstructure:"services"`
	Quota     Quota               `mapstructure:"quota"`
	JWT       JWT                 `mapstructure:"jwt"`
}

type App struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Tribe   string `mapstructure:"tribe"`
	Version string `mapstructure:"version"`
}

type Log struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type HTTP struct {
	Port            int `mapstructure:"port"`
	ReadTimeoutSec  int `mapstructure:"readTimeoutSec"`
	WriteTimeoutSec int `mapstructure:"writeTimeoutSec"`
	IdleTimeoutSec  int `mapstructure:"idleTimeoutSec"`
}

type APM struct {
	Enabled bool    `mapstructure:"enabled"`
	Host    string  `mapstructure:"host"`
	Port    int     `mapstructure:"port"`
	Rate    float64 `mapstructure:"rate"`
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Mongo struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

type OAuth struct {
	Google    OAuthClient `mapstructure:"google"`
	Microsoft OAuthClient `mapstructure:"microsoft"`
}

type OAuthClient struct {
	TenantID     string `mapstructure:"tenantID"`
	ClientID     string `mapstructure:"clientID"`
	ClientSecret string `mapstructure:"clientSecret"`
	RedirectURL  string `mapstructure:"redirectURL"`
}

// Provider is generic LLM-provider config. Adapters pick the fields they need.
type Provider struct {
	Enabled     bool              `mapstructure:"enabled"`
	BaseURL     string            `mapstructure:"baseURL"`
	APIKey      string            `mapstructure:"apiKey"`
	APIVersion  string            `mapstructure:"apiVersion"`
	Deployment  string            `mapstructure:"deployment"`
	TimeoutSec  int               `mapstructure:"timeoutSec"`
	ModelPrefix string            `mapstructure:"modelPrefix"` // optional auto-routing prefix, e.g. "claude-"
	Models      []string          `mapstructure:"models"`      // explicit model allowlist
	Headers     map[string]string `mapstructure:"headers"`
}

type ServiceCreds struct {
	Tribe    string `mapstructure:"tribe"`
	Name     string `mapstructure:"name"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type Quota struct {
	WindowSec   int `mapstructure:"windowSec"`
	TokensPerWindow int `mapstructure:"tokensPerWindow"`
}

type JWT struct {
	Secret       string `mapstructure:"secret"`
	ExpiresHours int    `mapstructure:"expiresHours"`
}

// Load reads config from path, applying APP_*-style env overrides.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config decode: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8080
	}
	if c.Quota.WindowSec == 0 {
		c.Quota.WindowSec = 3600
	}
	return nil
}
