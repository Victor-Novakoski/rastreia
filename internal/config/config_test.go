package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func validConfig() Config {
	return Config{
		DatabaseURL: "postgres://localhost/db",
		JWTSecret:   "a-real-secret-with-more-than-32-characters",
		AppEnv:      "production",
		AdminEmail:  "admin@example.com", AdminPassword: "a-strong-password",
		CORSOrigins:   "https://rastreia.dev, https://www.rastreia.dev",
		RetentionDays: 90,
	}
}

func TestValidate_RetentionAfterPublicLink(t *testing.T) {
	c := validConfig()
	c.RetentionDays = 30
	assert.ErrorContains(t, c.validate(), "RETENTION_DAYS")
}

func TestValidate(t *testing.T) {
	assert.NoError(t, validConfig().validate())

	cases := map[string]func(*Config){
		"missing database":     func(c *Config) { c.DatabaseURL = "" },
		"short secret":         func(c *Config) { c.JWTSecret = "short" },
		"dev secret in prod":   func(c *Config) { c.JWTSecret = devJWTSecret },
		"dev password in prod": func(c *Config) { c.AdminPassword = devAdminPassword },
		"http origin in prod":  func(c *Config) { c.CORSOrigins = "http://rastreia.dev" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validConfig()
			mutate(&c)
			assert.Error(t, c.validate())
		})
	}
}

func TestValidate_DevAllowsDefaults(t *testing.T) {
	c := validConfig()
	c.AppEnv = "development"
	c.JWTSecret = devJWTSecret
	c.AdminPassword = devAdminPassword
	c.CORSOrigins = "http://localhost:5173"
	assert.NoError(t, c.validate())
}

func TestAllowedOrigins(t *testing.T) {
	assert.Equal(t, []string{"https://a.dev", "https://b.dev"}, Config{CORSOrigins: " https://a.dev, ,https://b.dev "}.AllowedOrigins())
	assert.Nil(t, Config{}.AllowedOrigins())
}
