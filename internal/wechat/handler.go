package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	cfg *Config
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 兼容旧路由（前端已上线，不能改）
	mux.HandleFunc("/api/parse-wechat", h.parseWeChat)
	mux.HandleFunc("/api/import-wechat", h.parseWeChat)
	mux.HandleFunc("/api/download-media", h.downloadMedia)

	// 新路由（后续新服务用）
	mux.HandleFunc("/api/wechat/health", h.health)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, 405, map[string]string{"message": "Method not allowed"})
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"ok": true, "timestamp": time.Now().UTC().Format(time.RFC3339),
		"service": "wechat",
	})
}

type parseReq struct {
	ArticleURL string `json:"articleUrl"`
}

func (h *Handler) parseWeChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"message": "Method not allowed"})
		return
	}
	var req parseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"message": "Invalid request body"})
		return
	}
	req.ArticleURL = strings.TrimSpace(req.ArticleURL)
	if req.ArticleURL == "" {
		writeJSON(w, 400, map[string]string{"message": "articleUrl is required"})
		return
	}

	article, err := Parse(req.ArticleURL, h.cfg.FetchTimeout)
	if err != nil {
		log.Printf("[wechat] 解析失败: %v url=%s", err, req.ArticleURL)
		writeJSON(w, 500, map[string]string{"message": err.Error()})
		return
	}

	mediaSet, mediaURLs := map[string]struct{}{}, []string{}
	for _, u := range append(article.ImageURLs, article.VideoURLs...) {
		if _, exists := mediaSet[u]; !exists {
			mediaSet[u] = struct{}{}
			mediaURLs = append(mediaURLs, u)
		}
	}
	if len(mediaURLs) > h.cfg.MediaLimit {
		mediaURLs = mediaURLs[:h.cfg.MediaLimit]
	}

	writeJSON(w, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"articleUrl": req.ArticleURL,
			"title":      article.Title,
			"text":       article.Text,
			"textLength": len(article.Text),
			"imageUrls":  article.ImageURLs,
			"videoUrls":  article.VideoURLs,
			"mediaUrls":  mediaURLs,
			"imageCount": len(article.ImageURLs),
			"videoCount": len(article.VideoURLs),
			"mediaCount": len(mediaURLs),
		},
	})
}

type downloadReq struct {
	URL string `json:"url"`
}

func (h *Handler) downloadMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"message": "Method not allowed"})
		return
	}
	var req downloadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"message": "Invalid request body"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" || !strings.HasPrefix(req.URL, "http") {
		writeJSON(w, 400, map[string]string{"message": "valid media url is required"})
		return
	}

	client := &http.Client{Timeout: time.Duration(h.cfg.FetchTimeout) * time.Second}
	proxyReq, _ := http.NewRequest("GET", req.URL, nil)
	proxyReq.Header.Set("Referer", "https://mp.weixin.qq.com/")
	proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, 502, map[string]string{"message": "download failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		writeJSON(w, 502, map[string]string{"message": fmt.Sprintf("download failed from source: %d", resp.StatusCode)})
		return
	}

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		w.Header().Set("Content-Disposition", cd)
	}
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	w.Write(body)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
