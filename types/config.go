package types

const (
	AppName    = "s3Proxy"
	AppVersion = "0.0.1"
)

var Config struct {
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
