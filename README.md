# feishu-extension-services

飞书扩展统一后端服务，集中管理边栏插件、字段捷径、连接器等扩展的后端处理逻辑。

## 仓库信息

| 项目 | 信息 |
|:----|:------|
| 组织 | SpadeGo |
| GitHub | https://github.com/SpadeGo/feishu-extension-services |
| 服务器 | 175.24.181.14（掘金1号） |
| 容器 | Docker Compose 管理 |
| 域名 | fetch.jinchiyi.top（Nginx → 容器）|
| 端口 | 8787（容器内）|
| 技术栈 | Go 1.26 + Gin v1.12 |

## 当前已迁移的服务

| 服务 | 插件包 | 路由前缀 | Nginx 前缀 | 状态 |
|:----|:-------|:---------|:----------|:----:|
| 公众号解析 | `internal/wechat/` | `/api/wechat/` | `/wechat-importer-api/` | ✅ 已迁移 |
| 增值税发票识别 | `internal/invoice/` | `/api/invoice/` | `/extension-api/` | ✅ 已上线 |
| 抖音视频解析 | `internal/douyin/`（占位） | `/api/douyin/` | `/douyin-api/` | 🚧 待迁移 |

## Nginx 路由映射

```
/wechat-importer-api/api/* → nginx → feishu-extension-services:8787/api/*
/extension-api/api/*       → nginx → feishu-extension-services:8787/api/*
```

> Nginx 的 `proxy_pass` 尾部有斜杠时**去掉** location 前缀，保留剩余路径。

## 健康检查

```bash
curl https://fetch.jinchiyi.top/wechat-importer-api/api/health
# {"ok":true,"service":"feishu-extension-services"}
```

## 部署方式

### 方式一：本地编译 + docker cp（推荐，服务器 Go 1.26 环境）

```bash
# 1. 在服务器上拉取最新代码
ssh root@175.24.181.14
cd /root/code/feishu-extension-services
git checkout -- .
git pull origin main

# 2. 编译（CGO_ENABLED=0 生成 alpine 兼容的静态二进制）
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /tmp/server_alpine ./cmd/server/

# 3. 替换容器中的二进制并重启
docker cp /tmp/server_alpine feishu-extension-services:/app/server
docker compose -f /root/code/deploy/docker-compose.yml restart feishu-extension-services

# 4. 验证
docker exec feishu-extension-services wget -q -O- http://127.0.0.1:8787/api/health
```

### 方式二：Docker 构建（服务器网络好时可用）

```bash
docker build -t feishu-extension-services:latest .
docker compose -f /root/code/deploy/docker-compose.yml up -d feishu-extension-services
```

### 方式三：SCP 单文件同步（GitHub 拉取失败时）

```bash
# 本地
scp /path/to/internal/invoice/baidu.go root@175.24.181.14:/root/code/feishu-extension-services/internal/invoice/baidu.go

# 服务器
CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /tmp/server_alpine ./cmd/server/
docker cp /tmp/server_alpine feishu-extension-services:/app/server
docker compose -f /root/code/deploy/docker-compose.yml restart feishu-extension-services
```

### 网络问题处理

GitHub API 拉取不稳定时（Empty reply from server）：

```bash
git checkout -- .           # 丢弃本地修改
git pull origin main        # 重试 git pull
# 或直接用 SCP 同步改动的文件
```

## API

### 健康检查

```bash
GET /api/health
```

### 公众号解析

```bash
POST /api/wechat/parse-wechat
{"articleUrl": "https://mp.weixin.qq.com/s/..."}
```

### 增值税发票识别

```bash
POST /api/invoice/ocr
{"image_base64": "..."}   # 支持 JPEG/PNG/BMP/PDF/OFD
```

## 项目结构

```
feishu-extension-services/
├── cmd/server/main.go          # 统一入口（Gin 引擎 + 插件注册）
├── internal/
│   ├── core/                   # 核心框架
│   │   ├── plugin.go           # Plugin 接口
│   │   ├── server.go           # Server 封装
│   │   └── response.go         # 统一响应格式
│   ├── wechat/                 # 公众号解析
│   ├── invoice/                # 增值税发票 OCR
│   └── douyin/                 # 抖音视频解析（占位）
├── Dockerfile
├── go.mod / go.sum
└── README.md
```

## 快速启动（本地开发）

```bash
go run ./cmd/server/
# 监听 :8787
```
