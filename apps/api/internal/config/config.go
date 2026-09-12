package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                     string
	Port                    int
	Host                    string
	BaseURL                 string
	WebURL                  string
	DatabaseURL             string
	DBMaxOpenConns          int
	DBMaxIdleConns          int
	DBConnMaxLifetimeMins   int
	RedisURL                string
	RedisPassword           string
	SessionSecret           string
	SessionCookieName       string
	SessionCookieSecure     bool
	SessionCookieDomain     string
	SessionTTL              time.Duration
	GitRootDir              string
	GitDefaultBranch        string
	GitMaxDiffLines         int
	GitMaxDiffBytes         int64
	LogLevel                string
	LogFormat               string
}

func Load() *Config {
	return &Config{
		Env:                   getEnv("FORGEHUB_ENV", "development"),
		Port:                  getEnvAsInt("FORGEHUB_PORT", 8080),
		Host:                  getEnv("FORGEHUB_HOST", "0.0.0.0"),
		BaseURL:               getEnv("FORGEHUB_BASE_URL", "http://localhost:8080"),
		WebURL:                getEnv("FORGEHUB_WEB_URL", "http://localhost:5173"),
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		DBMaxOpenConns:        getEnvAsInt("DATABASE_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:        getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetimeMins: getEnvAsInt("DATABASE_CONN_MAX_LIFETIME_MINS", 30),
		RedisURL:              getEnv("REDIS_URL", ""),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		SessionSecret:         getEnv("SESSION_SECRET", "dev_session_secret_change_in_production_32bytes_min!"),
		SessionCookieName:     getEnv("SESSION_COOKIE_NAME", "forgehub_session"),
		SessionCookieSecure:   getEnvAsBool("SESSION_COOKIE_SECURE", false),
		SessionCookieDomain:   getEnv("SESSION_COOKIE_DOMAIN", "localhost"),
		SessionTTL:            time.Duration(getEnvAsInt("SESSION_TTL_HOURS", 72)) * time.Hour,
		GitRootDir:            getEnv("GIT_ROOT_DIR", "./data/git"),
		GitDefaultBranch:      getEnv("GIT_DEFAULT_BRANCH", "main"),
		GitMaxDiffLines:       getEnvAsInt("GIT_MAX_DIFF_LINES", 5000),
		GitMaxDiffBytes:       getEnvAsInt64("GIT_MAX_DIFF_BYTES", 5242880), // 5MB
		LogLevel:              strings.ToLower(getEnv("LOG_LEVEL", "info")),
		LogFormat:             strings.ToLower(getEnv("LOG_FORMAT", "json")),
	}
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Env) == "production"
}

func (c *Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}
