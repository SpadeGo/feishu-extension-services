package wechat

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port         int
	CORSOrigin   []string
	FetchTimeout int
	MediaLimit   int
}

func LoadConfig() *Config {
	return &Config{
		Port:         getEnvInt("WECHAT_PORT", 8787),
		CORSOrigin:   parseOrigins(getEnv("WECHAT_CORS_ORIGIN", "*")),
		FetchTimeout: getEnvInt("WECHAT_FETCH_TIMEOUT_SEC", 20),
		MediaLimit:   getEnvInt("WECHAT_MEDIA_LIMIT", 60),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func parseOrigins(origin string) []string {
	if origin == "" || origin == "*" {
		return []string{"*"}
	}
	parts := strings.Split(origin, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
