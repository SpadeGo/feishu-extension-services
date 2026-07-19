package invoice

import "time"

// Config 百度云 OCR 配置
type Config struct {
	APIKey      string        `mapstructure:"baidu_api_key"`
	SecretKey   string        `mapstructure:"baidu_secret_key"`
	TokenURL    string        `mapstructure:"baidu_token_url"`
	OCRURL      string        `mapstructure:"baidu_ocr_url"`
	TokenExpiry time.Duration `mapstructure:"baidu_token_expiry"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		APIKey:      "YbEWEmHvnR8uX9xhDYtdiADd",
		SecretKey:   "8ocBavy6D1CLyGB2nIRASNlzWb6ev1CP",
		TokenURL:    "https://aip.baidubce.com/oauth/2.0/token",
		OCRURL:      "https://aip.baidubce.com/rest/2.0/ocr/v1/vat_invoice",
		TokenExpiry: 24 * time.Hour, // 百度 access_token 有效期约 30 天，保守取 24h
	}
}
