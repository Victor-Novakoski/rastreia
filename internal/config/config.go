// Package config loads the application settings from environment variables.
package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port          string        `mapstructure:"PORT"`
	DatabaseURL   string        `mapstructure:"DATABASE_URL"`
	JWTSecret     string        `mapstructure:"JWT_SECRET"`
	JWTTTL        time.Duration `mapstructure:"JWT_TTL"`
	AdminName     string        `mapstructure:"ADMIN_NAME"`
	AdminEmail    string        `mapstructure:"ADMIN_EMAIL"`
	AdminPassword string        `mapstructure:"ADMIN_PASSWORD"`
}

// Load reads the configuration from the environment. A .env file in the
// working directory is used as a fallback when present.
func Load() (Config, error) {
	v := viper.New()
	v.SetDefault("PORT", "8080")
	v.SetDefault("JWT_TTL", "24h")
	v.SetDefault("ADMIN_NAME", "Admin")
	for _, key := range []string{"DATABASE_URL", "JWT_SECRET", "ADMIN_EMAIL", "ADMIN_PASSWORD"} {
		_ = v.BindEnv(key)
	}
	v.AutomaticEnv()

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig() // optional

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, cfg.validate()
}

func (c Config) validate() error {
	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must have at least 32 characters"))
	}
	return errors.Join(errs...)
}
