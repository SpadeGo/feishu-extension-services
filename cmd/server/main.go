package main

import (
	"os"

	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/SpadeGo/feishu-extension-services/internal/wechat"
	"github.com/gin-gonic/gin"
)

func main() {
	srv := core.New()

	// 全局 CORS 中间件
	srv.Use(corsMiddleware())

	// 注册各业务插件
	srv.Register(wechat.NewPlugin(wechat.LoadConfig()))
	// srv.Register(douyin.NewPlugin(...))  // 未来插件

	// 全局健康检查
	srv.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"ok":      true,
			"service": "feishu-extension-services",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}

	srv.Run(":" + port)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
