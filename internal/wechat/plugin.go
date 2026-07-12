package wechat

import (
	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/gin-gonic/gin"
)

// Plugin wraps the WeChat handler as a core.Plugin.
type Plugin struct {
	handler *Handler
}

// NewPlugin creates a new WeChat plugin with the given config.
func NewPlugin(cfg *Config) *Plugin {
	return &Plugin{handler: NewHandler(cfg)}
}

func (p *Plugin) Name() string { return "wechat" }

func (p *Plugin) RegisterRoutes(rg *gin.RouterGroup) {
	p.handler.RegisterRoutes(rg)
}

// Compile-time interface check.
var _ core.Plugin = (*Plugin)(nil)
