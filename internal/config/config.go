package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr              string
	BaseURL           string
	DataDir           string
	DBPath            string
	DBDriver          string // sqlite | libsql (turso-ready)
	DatabaseURL       string // for Turso/libsql later
	AdminUser         string
	AdminPassword     string
	SessionSecret     string
	TelegramBotToken  string
	TelegramChatID    string
	SecureCookies     bool
	FixturesPath      string
	StaticDir         string
	TemplatesDir      string
	MigrationsDir     string
	SeedOnEmpty       bool
}

func Load() Config {
	dataDir := getenv("DATA_DIR", "./data")
	return Config{
		Addr:             getenv("ADDR", ":3000"),
		BaseURL:          strings.TrimRight(getenv("BASE_URL", "http://localhost:3000"), "/"),
		DataDir:          dataDir,
		DBPath:           getenv("DB_PATH", dataDir+"/app.db"),
		DBDriver:         getenv("DB_DRIVER", "sqlite"),
		DatabaseURL:      getenv("DATABASE_URL", ""),
		AdminUser:        getenv("ADMIN_USER", "admin"),
		AdminPassword:    getenv("ADMIN_PASSWORD", "changeme"),
		SessionSecret:    getenv("SESSION_SECRET", "dev-session-secret-change-me"),
		TelegramBotToken: getenv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getenv("TELEGRAM_CHAT_ID", ""),
		SecureCookies:    getenvBool("SECURE_COOKIES", false),
		FixturesPath:     getenv("FIXTURES_PATH", "./fixtures/seed.json"),
		StaticDir:        getenv("STATIC_DIR", "./web/static"),
		TemplatesDir:     getenv("TEMPLATES_DIR", "./web/templates"),
		MigrationsDir:    getenv("MIGRATIONS_DIR", "./migrations"),
		SeedOnEmpty:      getenvBool("SEED_ON_EMPTY", true),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
