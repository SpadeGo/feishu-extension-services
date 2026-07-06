package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/SpadeGo/feishu-extension-services/internal/wechat"
)

func main() {
	mux := http.NewServeMux()

	// 公众号解析
	wxHandler := wechat.NewHandler(wechat.LoadConfig())
	wxHandler.RegisterRoutes(mux)

	// 注册全局路由
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"ok":true,"service":"feishu-extension-services"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8787"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: corsMiddleware(mux),
	}

	go func() {
		log.Printf("[main] 飞书扩展统一后端服务启动，监听 :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main] 服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[main] 正在关闭服务...")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
