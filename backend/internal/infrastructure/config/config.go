package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppName            string
	Version            string
	Addr               string
	DataDir            string
	MigrationsDir      string
	SeedsDir           string
	StaticDir          string
	PublicURL          string
	CookieSecure       bool
	SessionTTL         time.Duration
	UpdateCheckURL     string
	TokenEncryptionKey string
	LocalResetTokens   bool
	SMTP               SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	StartTLS bool
}

func Load(version string) Config {
	sessionDays := getenvInt("CALENDAR_SESSION_DAYS", 14)
	dataDir := getenv("CALENDAR_DATA_DIR", "./data")
	tokenKey := getenv("CALENDAR_TOKEN_ENCRYPTION_KEY", "")
	if tokenKey == "" {
		tokenKey = loadOrCreateLocalKey(dataDir)
	}
	publicURL := strings.TrimRight(getenv("CALENDAR_PUBLIC_URL", "http://localhost:8080"), "/")
	return Config{
		AppName:            "CalendarAdvanced",
		Version:            version,
		Addr:               getenv("CALENDAR_ADDR", ":8080"),
		DataDir:            dataDir,
		MigrationsDir:      getenv("CALENDAR_MIGRATIONS_DIR", "./migrations"),
		SeedsDir:           getenv("CALENDAR_SEEDS_DIR", "./seeds"),
		StaticDir:          getenv("CALENDAR_STATIC_DIR", "../frontend/dist"),
		PublicURL:          publicURL,
		CookieSecure:       getenvBool("CALENDAR_COOKIE_SECURE", false),
		SessionTTL:         time.Duration(sessionDays) * 24 * time.Hour,
		UpdateCheckURL:     getenv("CALENDAR_UPDATE_CHECK_URL", ""),
		TokenEncryptionKey: tokenKey,
		LocalResetTokens:   getenvBool("CALENDAR_LOCAL_RESET_TOKENS", false),
		SMTP: SMTPConfig{
			Host:     getenv("CALENDAR_SMTP_HOST", ""),
			Port:     getenvInt("CALENDAR_SMTP_PORT", 587),
			User:     getenv("CALENDAR_SMTP_USER", ""),
			Password: getenv("CALENDAR_SMTP_PASSWORD", ""),
			From:     getenv("CALENDAR_SMTP_FROM", ""),
			StartTLS: getenvBool("CALENDAR_SMTP_STARTTLS", true),
		},
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func getenvInt(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil && value > 0 {
		return value
	}
	return fallback
}

func loadOrCreateLocalKey(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return ""
	}
	path := dataDir + string(os.PathSeparator) + "calendaradvanced.key"
	if existing, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(existing))) >= 32 {
		return strings.TrimSpace(string(existing))
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	key := base64.StdEncoding.EncodeToString(buf)
	_ = os.WriteFile(path, []byte(key+"\n"), 0o600)
	return key
}
