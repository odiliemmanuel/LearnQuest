package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	OTPSecret   string
	RabbitMQURL string
	OpenAIKey   string
	OpenAIModel string
	AITimeout   time.Duration
	AIMock      bool
	AdminEmail  string
	IngestKey   string
	CORSOrigins []string

	// Embedded email consumer (see internal/notifications). Hosts without
	// background workers run it in-process.
	RunEmailConsumer bool
	SMTPHost         string
	SMTPPort         string
	SMTPUsername     string
	SMTPPassword     string
	EmailFrom        string
}

func Load() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://odili:postgres@localhost:5432/learnquest"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-only-change-me"),
		JWTTTL:      time.Duration(getEnvInt("JWT_TTL_MINUTES", 10080)) * time.Minute,
		OTPSecret:   os.Getenv("OTP_SECRET"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIModel: getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		AITimeout:   time.Duration(getEnvInt("AI_TIMEOUT_SECONDS", 20)) * time.Second,
		AIMock:      os.Getenv("OPENAI_API_KEY") == "",
		AdminEmail:  os.Getenv("ADMIN_EMAIL"),
		IngestKey:   getEnv("INGEST_SECRET", "dev-ingest-secret-change-me"),
		CORSOrigins: splitOrigins(getEnv("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),

		RunEmailConsumer: getEnv("RUN_EMAIL_CONSUMER", "") == "true",
		SMTPHost:         os.Getenv("SMTP_HOST"),
		SMTPPort:         getEnv("SMTP_PORT", "465"),
		SMTPUsername:     os.Getenv("SMTP_USERNAME"),
		SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
		EmailFrom:        os.Getenv("EMAIL_FROM"),
	}

	if cfg.JWTSecret == "dev-only-change-me" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}
	if cfg.OTPSecret == "" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("OTP_SECRET must be set in production")
	}
	if cfg.RabbitMQURL == "amqp://guest:guest@localhost:5672/" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("RABBITMQ_URL must be set in production")
	}
	if cfg.RunEmailConsumer && (cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" || cfg.EmailFrom == "") {
		return nil, fmt.Errorf("RUN_EMAIL_CONSUMER=true requires SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD and EMAIL_FROM")
	}
	if cfg.OTPSecret == "" {
		cfg.OTPSecret = cfg.JWTSecret
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitOrigins parses the comma-separated CORS_ORIGINS allowlist. Use "*" to
// allow any origin; credentials are still only sent to explicitly listed
// origins, since browsers reject "*" together with credentials.
func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadDotEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		path := dir + string(os.PathSeparator) + ".env"
		err := godotenv.Load(path)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
