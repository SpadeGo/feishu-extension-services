# feishu-extension-services

飞书扩展统一后端服务，集中管理边栏插件、字段捷径、连接器等扩展的后端处理逻辑。

## 当前已迁移的服务

| 服务 | 路由前缀 | 状态 |
|:----|:--------|:----:|
| 公众号解析 | `/api/wechat/` | ✅ 已迁移 |
| 抖音视频解析 | `/api/douyin/` | 🚧 待迁移 |
| 文案润色 | `/api/polish/` | 📅 计划中 |

## 快速启动

```bash
# 本地开发
go run ./cmd/server/

# 构建
go build -o server ./cmd/server/
./server
```

## Docker 部署

```bash
docker build -t feishu-extension-services .
docker run -d -p 8787:8787 feishu-extension-services
```

## API

### 公众号解析

```
POST /api/wechat/parse
{"articleUrl": "https://mp.weixin.qq.com/s/..."}

POST /api/wechat/download-media
{"url": "https://mmbiz.qpic.cn/..."}
```

### 迁移指南

将独立的后端服务迁移到本仓库：

1. 在 `internal/` 下创建子包，如 `internal/douyin/`
2. 实现 `RegisterRoutes(mux)` 方法注册路由
3. 在 `cmd/server/main.go` 中引入并注册
