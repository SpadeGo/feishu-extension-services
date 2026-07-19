package invoice

import (
	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/gin-gonic/gin"
)

// Plugin 发票识别插件
type Plugin struct {
	handler *Handler
}

// NewPlugin 创建发票识别插件
func NewPlugin(cfg *Config) *Plugin {
	return &Plugin{handler: NewHandler(cfg)}
}

func (p *Plugin) Name() string { return "invoice" }

func (p *Plugin) RegisterRoutes(rg *gin.RouterGroup) {
	p.handler.RegisterRoutes(rg)
}

// 编译时接口检查
var _ core.Plugin = (*Plugin)(nil)
