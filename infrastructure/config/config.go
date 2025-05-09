package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/rs/zerolog/log"
)

const (
	AppName    = "s3Proxy"
	AppVersion = "0.0.1"
)

// Config represents the application configuration
type Config struct {
	App struct {
		Listen     string `yaml:"listen" env:"LISTEN" env-default:":2804"`
		PPROF      string `yaml:"pprof" env:"PPROF"`
		LogLevel   int8   `yaml:"log_level" env:"LOG_LEVEL" env-default:"1"` // 0: debug, 1: info, 2: warning, 3: error, 4: fatal, 5: panic
		Sentry     string `yaml:"sentry" env:"SENTRY"`
		Cloudflare bool   `yaml:"cloudflare" env:"CLOUDFLARE" env-default:"true"`
		EnableList bool   `yaml:"enable_list" env:"ENABLE_LIST" env-default:"true"`
	} `yaml:"app"`

	S3 struct {
		Endpoint string `yaml:"endpoint" env:"S3_ENDPOINT" env-required:"true"`
		Bucket   string `yaml:"bucket" env:"S3_BUCKET" env-required:"true"`
		Key      struct {
			Access string `yaml:"access" env:"S3_ACCESS_KEY" env-required:"true"`
			Secret string `yaml:"secret" env:"S3_SECRET_KEY" env-required:"true"`
		} `yaml:"key"`
	} `yaml:"s3"`
}

// AppConfig is the global configuration instance
var AppConfig Config

// LoadConfig loads the configuration from environment variables
func LoadConfig() error {
	if err := cleanenv.ReadEnv(&AppConfig); err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		return err
	}
	return nil
}

// IsListEnabled returns whether the list endpoint is enabled
func IsListEnabled() bool {
	return AppConfig.App.EnableList
}

// GetListenAddress returns the configured listen address
func GetListenAddress() string {
	return AppConfig.App.Listen
}

// GetLogLevel returns the configured log level
func GetLogLevel() int8 {
	return AppConfig.App.LogLevel
}

// IsCloudflareEnabled returns whether Cloudflare integration is enabled
func IsCloudflareEnabled() bool {
	return AppConfig.App.Cloudflare
}

// GetPPROFPath returns the configured pprof path
func GetPPROFPath() string {
	return AppConfig.App.PPROF
}