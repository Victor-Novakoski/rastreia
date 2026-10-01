// Package config loads the application settings from environment variables.
package config

import (
	"errors"
	"strings"
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
	AppEnv        string        `mapstructure:"APP_ENV"`
	// CORSOrigins is a comma-separated list of front-end origins allowed to call the API.
	CORSOrigins string `mapstructure:"CORS_ORIGINS"`
	// TrustProxy makes the API read the client IP from X-Forwarded-For. Only
	// enable it behind a proxy that sets that header, or clients can fake
	// their IP and dodge the rate limit.
	TrustProxy bool `mapstructure:"TRUST_PROXY"`
}

// Values shipped in .env.example and docker-compose.yml. Production must override them.
const (
	devJWTSecret     = "dev-secret-change-me-at-least-32-chars"
	devAdminPassword = "admin12345"
)

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// AllowedOrigins splits CORSOrigins, ignoring blanks.
func (c Config) AllowedOrigins() []string {
	var out []string
	for _, o := range strings.Split(c.CORSOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// Load reads the configuration from the environment. A .env file in the
// working directory is used as a fallback when present.
func Load() (Config, error) {
	v := viper.New()
	v.SetDefault("PORT", "8080")
	v.SetDefault("JWT_TTL", "24h")
	v.SetDefault("ADMIN_NAME", "Admin")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("CORS_ORIGINS", "http://localhost:5173")
	v.SetDefault("TRUST_PROXY", false)
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
	if c.IsProduction() {
		if c.JWTSecret == devJWTSecret {
			errs = append(errs, errors.New("JWT_SECRET must not use the development value in production"))
		}
		if c.AdminEmail != "" && c.AdminPassword == devAdminPassword {
			errs = append(errs, errors.New("ADMIN_PASSWORD must not use the development value in production"))
		}
		for _, o := range c.AllowedOrigins() {
			if !strings.HasPrefix(o, "https://") {
				errs = append(errs, errors.New("CORS_ORIGINS must only list https origins in production"))
				break
			}
		}
	}
	return errors.Join(errs...)
}
