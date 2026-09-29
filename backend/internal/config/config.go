package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		CORSOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
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
