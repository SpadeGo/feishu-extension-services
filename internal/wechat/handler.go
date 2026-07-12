package wechat

import (
	"log"
	"strings"

	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/gin-gonic/gin"
)

// Plugin-specific error codes.
const (
	CodeBadRequest  = 10001 // 请求参数错误
	CodeParseFailed = 10002 // 公众号文章解析失败
)

type Handler struct {
	cfg *Config
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/parse-wechat", h.parseWeChat)
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
