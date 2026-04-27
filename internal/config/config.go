package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port                string
	Env                 string
	CORSOrigins         string
	StaticCacheDuration time.Duration

	// PocketBase
	PBUrl   string
	PBAdmin string
	PBPass  string

	// Admin credentials
	AdminEmail    string
	AdminPassword string

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// Google OAuth2
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Cloudflare R2
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2Region     string
	R2PublicURL  string

	// WhatsApp (Twilio)
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Ollama
	OllamaURL   string
	OllamaModel string

	// Web
	BaseURL  string
	SiteName string
}

func Load() *Config {
	cfg := &Config{
		// Server
		Port:                getEnv("PORT", "3000"),
		Env:                 getEnv("ENV", "development"),
		CORSOrigins:         getEnv("CORS_ORIGINS", "*"),
		StaticCacheDuration: 24 * time.Hour,

		// PocketBase
		PBUrl:   getEnv("PB_URL", "http://127.0.0.1:8090"),
		PBAdmin: getEnv("PB_ADMIN_EMAIL", "admin@jcp.cl"),
		PBPass:  getEnv("PB_ADMIN_PASSWORD", ""),

		// Admin credentials — no hardcoded defaults for secrets
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@jcp-gestioninmobiliaria.cl"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),

		// JWT — no hardcoded default
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: 72 * time.Hour,

		// Google OAuth2
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:3000/auth/google/callback"),

		// Cloudflare R2
		R2AccountID:  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName: getEnv("R2_BUCKET_NAME", "jcp-media"),
		R2Region:     getEnv("R2_REGION", "auto"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", ""),

		// WhatsApp
		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),

		// Ollama
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "llama3"),

		// Web
		BaseURL:  getEnv("BASE_URL", "http://localhost:3000"),
		SiteName: "JCP Gestión Inmobiliaria",
	}

	// In development only, provide safe placeholder defaults so the app
	// starts without any env file. These values must never reach production.
	if !cfg.IsProd() {
		if cfg.AdminPassword == "" {
			cfg.AdminPassword = "dev-only-change-me"
		}
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "dev-jwt-secret-change-me"
		}
	}

	return cfg
}

// IsProd returns true when running in the production environment.
func (c *Config) IsProd() bool {
	return c.Env == "production"
}

// ValidateRequired aborts startup if critical secrets are missing in production.
func ValidateRequired(cfg *Config) {
	if !cfg.IsProd() {
		return
	}
	var missing []string
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.AdminPassword == "" {
		missing = append(missing, "ADMIN_PASSWORD")
	}
	if len(missing) > 0 {
		log.Fatalf("FATAL: variables requeridas en producción no configuradas: %s\n"+
			"Configuralas con: fly secrets set %s=<valor>",
			strings.Join(missing, ", "),
			strings.Join(missing, "=<valor> "),
		)
	}
	if cfg.CORSOrigins == "*" {
		fmt.Println("ADVERTENCIA: CORS_ORIGINS es '*' en producción. Considerá restringirlo.")
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
