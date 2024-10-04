package config

var Service = "proxy-service"
var Version = "v1.0.0"
var GitCommit string
var OSBuildName string
var BuildDate string

type MainConfig struct {
	Log struct {
		Level  string `yaml:"level" validate:"oneof=trace debug info warn error fatal panic"`
		Format string `yaml:"format" validate:"oneof=text json"`
	} `yaml:"log"`
	APM struct {
		Enabled bool     `yaml:"enabled"`
		Host    string   `yaml:"host"`
		Port    int      `yaml:"port" validate:"required,min=1,max=65535"`
		Rate    *float64 `yaml:"rate" validate:"omitempty,min=0.1,max=1"`
	} `yaml:"apm"`
	App struct {
		Name    string `yaml:"name" validate:"required"`
		Port    int    `yaml:"port" validate:"required,min=1,max=65535"`
		Version string `yaml:"version" validate:"required"`
		Env     string `yaml:"env" validate:"required"`
		Tribe   string `yaml:"tribe" validate:"required"`
	}
	Redis struct {
		Host     string `yaml:"host" validate:"required"`
		Port     int    `yaml:"port" validate:"required"`
		Password string `yaml:"password" validate:"required"`
		DB       int    `yaml:"db"`
	}
	Mongo struct {
		Host     string `yaml:"host" validate:"required"`
		Port     int    `yaml:"port" validate:"required"`
		Username string `yaml:"username" validate:"required"`
		Password string `yaml:"password" validate:"required"`
		DB       string `yaml:"db" validate:"required"`
	}
	UI struct {
		Host string `yaml:"host" validate:"required"`
	} `yaml:"ui"`
	MicrosoftOauth struct {
		TenantID     string `yaml:"tenantID" validate:"required"`
		ClientID     string `yaml:"clientID" validate:"required"`
		ClientSecret string `yaml:"clientSecret" validate:"required"`
		RedirectURL  string `yaml:"redirectURL" validate:"required"`
	} `yaml:"microsoftOauth"`
	GoogleOauth struct {
		ClientID     string `yaml:"clientID" validate:"required"`
		ClientSecret string `yaml:"clientSecret" validate:"required"`
		RedirectURL  string `yaml:"redirectURL" validate:"required"`
	} `yaml:"googleOauth"`
	OpenAI struct {
		Host          string `yaml:"host" validate:"required"`
		Path          string `yaml:"path" validate:"required"`
		ApiKey        string `yaml:"apiKey" validate:"required"`
		TokenLifetime int    `yaml:"tokenLifetime" validate:"required"`
		TokenLimit    int    `yaml:"tokenLimit" validate:"required"`
	} `yaml:"openAI"`
	Services []BackendService `yaml:"services"`
}

type BackendService struct {
	Tribe    string `yaml:"tribe"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
