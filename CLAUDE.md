# feishu-extension-services — Claude Code 部署手册

## 项目信息

- **组织**: SpadeGo
- **GitHub**: https://github.com/SpadeGo/feishu-extension-services
- **服务器**: `175.24.181.14`（掘金1号），root 密码 `Lyf.159??zwj`
- **容器名**: `feishu-extension-services`
- **容器端口**: 8787
- **Docker Compose**: `/root/code/deploy/docker-compose.yml`
- **域名**: `fetch.jinchiyi.top`
- **技术栈**: Go 1.26 / Gin v1.12

## 部署命令（一键）

```bash
sshpass -p 'Lyf.159??zwj' ssh -o StrictHostKeyChecking=no root@175.24.181.14 \
  'cd /root/code/feishu-extension-services && \
   git checkout -- . && \
   git pull origin main 2>&1 && \
   CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /tmp/server_alpine ./cmd/server/ 2>&1 && \
   echo "BUILD_OK: $(stat -c%s /tmp/server_alpine) bytes" && \
   docker cp /tmp/server_alpine feishu-extension-services:/app/server && \
   docker compose -f /root/code/deploy/docker-compose.yml restart feishu-extension-services && \
   echo "DEPLOY_OK"'
```

## 单文件同步（GitHub 拉取失败时）

```bash
# 本地 SCP
sshpass -p 'Lyf.159??zwj' scp ./internal/invoice/xxx.go root@175.24.181.14:/root/code/feishu-extension-services/internal/invoice/xxx.go

# 服务器编译部署
sshpass -p 'Lyf.159??zwj' ssh root@175.24.181.14 \
  'cd /root/code/feishu-extension-services && \
   CGO_ENABLED=0 go build -a -ldflags="-s -w" -o /tmp/server_alpine ./cmd/server/ && \
   docker cp /tmp/server_alpine feishu-extension-services:/app/server && \
   docker compose -f /root/code/deploy/docker-compose.yml restart feishu-extension-services'
```

## 验证

```bash
# 健康检查
curl -s https://jinchiyi.top/extension-api/api/health
# 预期: {"ok":true,"service":"feishu-extension-services"}

# 发票识别测试
python3 -c "
import requests, base64
with open('/tmp/unicom_invoice.jpg','rb') as f:
    b64 = base64.b64encode(f.read()).decode()
r = requests.post('https://jinchiyi.top/extension-api/api/invoice/ocr',
    json={'image_base64':b64}, timeout=30)
print(r.json()['code'], r.json()['message'])
"
```

## 注意事项

- **CGO_ENABLED=0** 必须设置，否则 alpine 容器运行报错
- **容器重启后需重新连 network**：`docker network connect code_app-network feishu-extension-services`（如果 Nginx 和本服务不在同一网络）
- **git pull 超时**：GitHub API 不稳定时用 SCP 单文件同步
- **Go 版本**：服务器 Go 1.26（`/opt/homebrew/opt/go@1.26/bin/go`），用系统默认 `go` 命令即可
