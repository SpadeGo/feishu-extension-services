package wechat

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/gin-gonic/gin"
)

// Plugin-specific error codes.
const (
	CodeBadRequest  = 10001 // 请求参数错误
	CodeParseFailed = 10002 // 公众号文章解析失败
	CodeProxyFailed = 10003 // 媒体代理拉取失败
)

type Handler struct {
	cfg *Config
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/parse-wechat", h.parseWeChat)
	rg.GET("/proxy-image", h.proxyImage)
}

type parseReq struct {
	ArticleURL string `json:"articleUrl"`
}

func (h *Handler) parseWeChat(c *gin.Context) {
	var req parseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		core.Fail(c, CodeBadRequest, "Invalid request body")
		return
	}
	req.ArticleURL = strings.TrimSpace(req.ArticleURL)
	if req.ArticleURL == "" {
		core.Fail(c, CodeBadRequest, "articleUrl is required")
		return
	}

	article, err := Parse(req.ArticleURL, h.cfg.FetchTimeout)
	if err != nil {
		log.Printf("[wechat] 解析失败: %v url=%s", err, req.ArticleURL)
		core.Fail(c, CodeParseFailed, err.Error())
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

	core.Success(c, gin.H{
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
	})
}

func (h *Handler) proxyImage(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		core.Fail(c, CodeBadRequest, "url is required")
		return
	}
	if !strings.HasPrefix(imageURL, "http") {
		core.Fail(c, CodeBadRequest, "invalid url")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		core.Fail(c, CodeProxyFailed, "failed to create request")
		return
	}
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		core.FailWithStatus(c, 502, CodeProxyFailed, "proxy failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		core.FailWithStatus(c, 502, CodeProxyFailed, "upstream returned "+http.StatusText(resp.StatusCode))
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	c.Data(200, contentType, body)
}
