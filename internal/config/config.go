package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration.
type Config struct {
	// Simap API
	SimapBaseURL string // default: https://www.simap.ch
	LookbackDays int    // how many days back to scan (default: 1)

	// Ollama (local LLM)
	OllamaBaseURL string // default: http://localhost:11434
	OllamaModel   string // default: mistral

	// SMTP / mail
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	MailFrom     string
	MailTo       string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		SimapBaseURL:  getEnv("SIMAP_BASE_URL", "https://www.simap.ch"),
		LookbackDays:  getEnvInt("LOOKBACK_DAYS", 1),
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "mistral"),
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		MailFrom:      getEnv("MAIL_FROM", ""),
		MailTo:        getEnv("MAIL_TO", ""),
	}

	var missing []string
	if cfg.SMTPHost == "" {
		missing = append(missing, "SMTP_HOST")
	}
	if cfg.MailFrom == "" {
		missing = append(missing, "MAIL_FROM")
	}
	if cfg.MailTo == "" {
		missing = append(missing, "MAIL_TO")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		fmt.Sscanf(v, "%d", &i)
		if i > 0 {
			return i
		}
	}
	return fallback
}
