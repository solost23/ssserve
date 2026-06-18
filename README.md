# ssserver

基于 Docker Compose 的 Shadowsocks 服务，包含多用户 token 管理、动态端口分配、流量统计和 Clash 订阅。

## 架构

- **ssserver**：shadowsocks-rust，Manager 模式；每个用户独立端口和密码。
- **subserver**：Go 管理服务；通过 Manager API 创建/吊销用户端口，生成 Clash YAML。
- **nginx**：提供 HTTP 管理界面和订阅接口。

## 快速部署

```bash
cp .env.example .env
./setup.sh
docker compose up -d --build
```

`.env` 只需要填：

```env
SERVER_ADDR=38.47.105.9
SS_CIPHER=chacha20-ietf-poly1305
SS_NAME=Tokyo
SS_USER_PORT_START=40200
ADMIN_SECRET=replace-with-a-long-random-secret
MANAGER_ADDR=ssserver:6001
```

`SERVER_ADDR` 会同时用于：

- 管理界面地址：`http://SERVER_ADDR/`
- 订阅链接：`http://SERVER_ADDR/sub/<token>/clash.yaml`
- Clash 节点里的 `server`

生成 `ADMIN_SECRET`：

```bash
openssl rand -hex 32
```

## 管理操作

- 创建订阅：在管理界面填写名称，点击 `Create`。
- 电脑 Clash 使用 `Clash` 链接。
- 手机 Shadowsocks 使用 `SS` 链接。
- 链接不存数据库，会根据 token、密码、端口和 `SERVER_ADDR` 动态生成。
- 吊销订阅：点击 `Revoke`，订阅立即失效。
- 查看流量：管理界面会显示每个 token 的已用流量和配额进度。

## 重新部署

只重新生成配置并启动：

```bash
./setup.sh
docker compose up -d --build
```

完全重置运行配置但保留数据库：

```bash
./cleanup.sh
./setup.sh
docker compose up -d --build
```

完全重置，包括删除所有 token 和统计数据：

```bash
./cleanup.sh --with-db
./setup.sh
docker compose up -d --build
```

## 验证

```bash
docker compose ps
curl -I http://127.0.0.1/
curl -I http://SERVER_ADDR/
docker logs --tail 100 nginx-sub
```

查看订阅访问日志：

```bash
docker exec nginx-sub tail -f /var/log/nginx/sub.access.log
```

## 文件说明

仓库只提交模板文件，真实配置、密码不进 git。

| 文件 | 说明 |
|------|------|
| `.env.example` | 配置变量模板 |
| `config/config.example.json` | ssserver 配置模板 |
| `config/nginx.example.conf` | nginx 主配置模板 |
| `server/` | Go 管理服务源码 |
| `web/index.html` | 前端管理界面 |
| `setup.sh` | 从 `.env` 生成运行时配置 |
| `cleanup.sh` | 清除生成的运行时配置 |

## 安全说明

- 每个用户有独立 ss 密码，互不影响。
- 订阅链接含随机 token，泄露后应立即吊销并重新创建。
- `ADMIN_SECRET` 是管理界面的唯一认证，请设置为强随机值。
- 订阅内容包含节点端口和密码，建议仅在可信网络内使用 HTTP 明文订阅。
