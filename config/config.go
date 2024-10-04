package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config interface {
	Init(configPath string) error
	Get() *MainConfig
}

type config struct {
	Config *MainConfig
}

func New() Config {
	return &config{
		Config: &MainConfig{},
	}
}

func (c *config) Init(configPath string) error {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		// Log the error if needed, but continue to load other configurations
	}

	if err := c.load(c.Config, ".", configPath); err != nil {
		return err
	}
	err := validator.New().Struct(c.Config)
	if err != nil {
		return err
	}

	return nil
}

func (c *config) Get() *MainConfig {
	return c.Config
}

func (c *config) load(cfg *MainConfig, path string, configPath string) error {
	// Set default values
	viper.SetDefault("log.level", "info")

	viper.AddConfigPath(path)
	if configPath != "" {
		viper.SetConfigFile(configPath)
	}
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()

	// Read the config file
	if err := viper.ReadInConfig(); err != nil && configPath != "" {
		return err
	}

	// Unmarshal the config into the struct
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	// Populate struct from environment variables using reflection
	if err := c.populateFromEnv(cfg); err != nil {
		return err
	}

	return nil
}

// populateFromEnv populates struct fields from environment variables if the `env` tag is present
func (c *config) populateFromEnv(cfg any) error {
	val := reflect.ValueOf(cfg)

	// Ensure that the value is a pointer to a struct
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a pointer but got %v", val.Kind())
	}

	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to struct but got %v", val.Elem().Kind())
	}

	val = val.Elem()  // Dereference the pointer to get to the struct
	typ := val.Type() // Get the struct type

	return c.populate(val, typ)
}

// populate recursively populates struct fields from environment variables if the `env` tag is present
func (c *config) populate(val reflect.Value, typ reflect.Type) error {
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		// Get the 'env' tag
		envTag := structField.Tag.Get("env")
		if envTag != "" {
			// Get the environment variable value from Viper
			envValue := viper.GetString(envTag)
			if envValue != "" {
				// Set the field based on its type
				switch field.Kind() {
				case reflect.Ptr:
					if field.Type().Elem().Kind() == reflect.String {
						field.Set(reflect.ValueOf(&envValue)) // set *string
					} else if field.Type().Elem().Kind() == reflect.Int {
						// Convert string to int before setting if it's an integer field
						var intVal int
						fmt.Sscanf(envValue, "%d", &intVal)
						field.Set(reflect.ValueOf(&intVal)) // set *int
					}
				case reflect.String:
					field.SetString(envValue) // set string
				case reflect.Int:
					// Convert string to int before setting if it's an integer field
					var intVal int
					fmt.Sscanf(envValue, "%d", &intVal)
					field.SetInt(int64(intVal)) // set int
				case reflect.Bool:
					// Convert string to bool before setting if it's a boolean field
					var boolVal bool
					fmt.Sscanf(envValue, "%t", &boolVal)
					field.SetBool(boolVal) // set bool
				}
			}
		}

		// Check if the field is a struct
		if field.Kind() == reflect.Struct {
			// Call populate recursively for nested structs
			if err := c.populate(field, field.Type()); err != nil {
				return err
			}
		}
	}

	return nil
}
