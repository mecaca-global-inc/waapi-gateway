package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr      string
	DBDialect     string
	DBURI         string
	AdminUser     string
	AdminPass     string
	LogLevel      string
	LogFormat     string // "console" (default) or "json"
	MediaDir      string
	CORSOrigins   []string
	AllowWeakAuth bool // set ALLOW_WEAK_AUTH=1 to bypass admin/admin guard
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	c := &Config{
		HTTPAddr:      env("HTTP_ADDR", ":3000"),
		DBDialect:     env("DB_DIALECT", "sqlite3"),
		DBURI:         env("DB_URI", "file:storages/gateway.db?_foreign_keys=on"),
		AdminUser:     env("ADMIN_USER", "admin"),
		AdminPass:     env("ADMIN_PASS", "admin"),
		LogLevel:      env("LOG_LEVEL", "info"),
		LogFormat:     env("LOG_FORMAT", "console"),
		MediaDir:      env("MEDIA_DIR", "storages/media"),
		CORSOrigins:   strings.Split(env("CORS_ORIGINS", "*"), ","),
		AllowWeakAuth: env("ALLOW_WEAK_AUTH", "") == "1",
	}
	if c.DBDialect != "sqlite3" && c.DBDialect != "postgres" {
		return nil, fmt.Errorf("unsupported DB_DIALECT %q (sqlite3|postgres)", c.DBDialect)
	}
	if !c.AllowWeakAuth {
		weak := map[string]bool{"admin": true, "changeme": true, "password": true, "": true}
		if weak[c.AdminPass] {
			return nil, fmt.Errorf("ADMIN_PASS is weak (%q). Set a strong password, or pass ALLOW_WEAK_AUTH=1 for local dev only", c.AdminPass)
		}
	}
	if err := os.MkdirAll("storages", 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.MediaDir, 0o755); err != nil {
		return nil, err
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func EnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
