package core

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

// Server wraps the Gin engine and manages plugin registration,
// global middleware, and graceful shutdown.
type Server struct {
	engine  *gin.Engine
	plugins []Plugin
}

// New creates a Server with Gin's default middleware (Logger and Recovery).
func New() *Server {
	r := gin.Default()
	return &Server{engine: r}
}

// Use registers global middleware on the Gin engine.
func (s *Server) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

// Register adds a plugin to the server. The plugin's routes are
// registered under the /api prefix.
func (s *Server) Register(p Plugin) {
	s.plugins = append(s.plugins, p)
	api := s.engine.Group("/api")
	p.RegisterRoutes(api)
	log.Printf("[core] registered plugin: %s", p.Name())
}

// GET is a thin wrapper for registering top-level routes
// (e.g. health checks) directly on the engine.
func (s *Server) GET(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.GET(relativePath, handlers...)
}

// Run starts the HTTP server on the given address (e.g. ":8787")
// with graceful shutdown on SIGINT/SIGTERM.
func (s *Server) Run(addr string) {
	server := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	go func() {
		log.Printf("[core] feishu-extension-services starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[core] server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[core] shutting down...")
}
