// Package core provides the plugin system and server bootstrap for
// feishu-extension-services. Each functional module (wechat, douyin, etc.)
// implements the Plugin interface and registers its routes via RegisterRoutes.
package core

import "github.com/gin-gonic/gin"

// Plugin is the interface that every backend plugin must implement.
// Plugins are registered with the Server and receive a *gin.RouterGroup
// scoped to the /api path prefix to register their HTTP routes.
type Plugin interface {
	// Name returns a human-readable name for the plugin (e.g. "wechat").
	Name() string

	// RegisterRoutes registers the plugin's HTTP handlers on the given
	// router group. The group is pre-scoped to /api — routes should be
	// registered as relative paths (e.g. "/parse-wechat").
	RegisterRoutes(rg *gin.RouterGroup)
}
