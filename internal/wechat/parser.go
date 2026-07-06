package wechat

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Article struct {
	Title     string   `json:"title"`
	Text      string   `json:"text"`
	ImageURLs []string `json:"imageUrls"`
	VideoURLs []string `json:"videoUrls"`
}

func Parse(articleURL string, timeoutSec int) (*Article, error) {
	if !strings.Contains(articleURL, "mp.weixin.qq.com") {
		return nil, fmt.Errorf("仅支持 mp.weixin.qq.com 链接")
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, _ := http.NewRequest("GET", articleURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取文章失败，状态码: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	title := extractTitle(doc)
	contentRoot := doc.Find("#js_content")
	if contentRoot.Length() == 0 {
		return nil, fmt.Errorf("未找到文章内容 (#js_content)")
	}

	return &Article{
		Title:     title,
		Text:      extractText(contentRoot),
		ImageURLs: extractImages(contentRoot, articleURL),
		VideoURLs: extractVideos(contentRoot, articleURL),
	}, nil
}

func extractTitle(doc *goquery.Document) string {
	if t := strings.TrimSpace(doc.Find("#activity-name").Text()); t != "" {
		return t
	}
	if t, ok := doc.Find(`meta[property="og:title"]`).Attr("content"); ok {
		if t = strings.TrimSpace(t); t != "" {
			return t
		}
	}
	return ""
}

func extractText(root *goquery.Selection) string {
	clone := root.Clone()
	clone.Find("script, style, iframe").Remove()

	var images []string
	clone.Find("img[data-src]").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("data-src"); ok && src != "" {
			images = append(images, src)
		}
	})
	clone.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok && src != "" && !strings.HasPrefix(src, "data:") {
			images = append(images, src)
		}
	})

	text := strings.TrimSpace(clone.Text())
	if text == "" {
		if len(images) > 0 {
			text = "[图片内容]"
		}
	}

	return text
}

func extractImages(root *goquery.Selection, baseURL string) []string {
	var urls []string
	seen := map[string]struct{}{}

	root.Find("img[data-src]").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("data-src"); ok {
			src = strings.TrimSpace(src)
			if src != "" {
				if _, exists := seen[src]; !exists {
					seen[src] = struct{}{}
					urls = append(urls, src)
				}
			}
		}
	})

	root.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			src = strings.TrimSpace(src)
			if src != "" && !strings.HasPrefix(src, "data:") {
				if _, exists := seen[src]; !exists {
					seen[src] = struct{}{}
					urls = append(urls, src)
				}
			}
		}
	})

	return urls
}

func extractVideos(root *goquery.Selection, baseURL string) []string {
	var urls []string
	seen := map[string]struct{}{}

	root.Find("video").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			src = strings.TrimSpace(src)
			if src != "" {
				if _, exists := seen[src]; !exists {
					seen[src] = struct{}{}
					urls = append(urls, src)
				}
			}
		}
	})

	root.Find("mpvideo").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			src = strings.TrimSpace(src)
			if src != "" {
				if _, exists := seen[src]; !exists {
					seen[src] = struct{}{}
					urls = append(urls, src)
				}
			}
		}
	})

	return urls
}
