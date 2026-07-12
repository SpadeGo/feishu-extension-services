package wechat

import (
	"os"
	"strconv"
)

// Config holds WeChat-specific configuration.
type Config struct {
	FetchTimeout int // HTTP fetch timeout in seconds
	MediaLimit   int // Max media URLs returned per request
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		FetchTimeout: getEnvInt("WECHAT_FETCH_TIMEOUT_SEC", 20),
		MediaLimit:   getEnvInt("WECHAT_MEDIA_LIMIT", 60),
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
