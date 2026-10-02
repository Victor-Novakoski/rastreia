package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

// Worker holds the settings of the notification worker (cmd/worker).
type Worker struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	RabbitMQURL string `mapstructure:"RABBITMQ_URL"`
	// TrackingURL is the front's public tracking page; e-mails link to
	// TrackingURL/<code>.
	TrackingURL  string `mapstructure:"TRACKING_URL"`
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     int    `mapstructure:"SMTP_PORT"`
	SMTPUsername string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`
	SMTPTLS      bool   `mapstructure:"SMTP_TLS"`
	// Web Push runs when the VAPID keys are set. It needs the database to
	// read the browsers that subscribed. Generate keys with: worker vapid
	DatabaseURL     string `mapstructure:"DATABASE_URL"`
	VAPIDPublicKey  string `mapstructure:"VAPID_PUBLIC_KEY"`
	VAPIDPrivateKey string `mapstructure:"VAPID_PRIVATE_KEY"`
	VAPIDSubject    string `mapstructure:"VAPID_SUBJECT"`
}

// PushEnabled tells whether the worker sends Web Push.
func (c Worker) PushEnabled() bool {
	return c.VAPIDPrivateKey != ""
}

// LoadWorker reads the worker settings, with .env as a fallback like Load.
func LoadWorker() (Worker, error) {
	v := viper.New()
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("TRACKING_URL", "http://localhost:5173/rastreio")
	v.SetDefault("SMTP_HOST", "localhost")
	v.SetDefault("SMTP_PORT", 1025)
	v.SetDefault("SMTP_FROM", "Rastreia <nao-responda@rastreia.dev>")
	v.SetDefault("SMTP_TLS", false)
	v.SetDefault("VAPID_SUBJECT", "mailto:contato@rastreia.dev")
	for _, key := range []string{"RABBITMQ_URL", "SMTP_USERNAME", "SMTP_PASSWORD", "DATABASE_URL", "VAPID_PUBLIC_KEY", "VAPID_PRIVATE_KEY"} {
		_ = v.BindEnv(key)
	}
	v.AutomaticEnv()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig() // optional

	var cfg Worker
	if err := v.Unmarshal(&cfg); err != nil {
		return Worker{}, err
	}
	return cfg, cfg.validate()
}

func (c Worker) validate() error {
	var errs []error
	if c.RabbitMQURL == "" {
		errs = append(errs, errors.New("RABBITMQ_URL is required"))
	}
	if c.PushEnabled() && (c.VAPIDPublicKey == "" || c.DatabaseURL == "") {
		errs = append(errs, errors.New("VAPID_PRIVATE_KEY needs VAPID_PUBLIC_KEY and DATABASE_URL for Web Push"))
	}
	if c.AppEnv == "production" {
		if !c.SMTPTLS {
			errs = append(errs, errors.New("SMTP_TLS must be true in production"))
		}
		if !strings.HasPrefix(c.TrackingURL, "https://") {
			errs = append(errs, errors.New("TRACKING_URL must be https in production"))
		}
	}
	return errors.Join(errs...)
}
