# xray-sub

基于 Docker Compose 的 Xray VLESS REALITY 服务，包含用户 token 管理、动态 UUID 下发、流量统计、配额控制和 Clash Meta 订阅。

## 架构

- **xray**：VLESS + REALITY + Vision，公网只暴露一个 TCP 入口。
- **subserver**：Go 管理服务，通过 Xray gRPC API 创建、暂停、恢复和删除用户 UUID。
- **nginx**：提供管理界面、API 反向代理和订阅接口。
- **web/front**：Vue 3 管理界面。

## 准备

需要 Docker 和 Docker Compose。

复制配置模板：

```bash
cp .env.example .env
```

生成 REALITY 密钥、short id 和管理密码：

```bash
docker run --rm ghcr.io/xtls/xray-core x25519
openssl rand -hex 8
openssl rand -hex 32
```

把生成结果填入 `.env`。至少需要这些变量：

```env
SERVER_ADDR=38.47.105.9
NODE_NAME=Tokyo
ADMIN_SECRET=replace-with-a-long-random-secret
XRAY_PORT=443
XRAY_PRIVATE_KEY=replace-with-reality-private-key
XRAY_PUBLIC_KEY=replace-with-reality-public-key
XRAY_SHORT_ID=replacehex
XRAY_SERVER_NAME=www.microsoft.com
XRAY_DEST=www.microsoft.com:443
```

`SERVER_ADDR` 会用于：

- 管理界面地址：`http://SERVER_ADDR/`
- 订阅链接：`http://SERVER_ADDR/sub/<token>/clash.yaml`
- 客户端节点里的 `server`

## 生成配置

运行：

```bash
./setup.sh
```

它会根据 `.env` 生成运行时配置：

- `config/nginx.conf`
- `config/xray.json`

`config/xray.json` 是 Docker Compose 挂载给 Xray 的实际配置文件：

```yaml
./config/xray.json:/etc/xray/config.json:ro
```

这个文件包含 REALITY 私钥，所以不会提交到 git。新机器部署时，本地没有 `config/xray.json` 是正常的，先运行 `./setup.sh` 生成它。

## 启动

```bash
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
docker logs --tail 100 xray
docker logs --tail 100 subserver
```

访问管理界面：

```text
http://SERVER_ADDR/
```

首次登录使用 `.env` 里的 `ADMIN_SECRET`。

## 管理使用

- 创建订阅：在管理界面填写名称，点击 `Create`。
- Clash Meta：使用 `Clash` 订阅链接。
- sing-box / v2rayN 等客户端：使用 `VLESS` 链接。
- 暂停或恢复用户：在管理界面操作，后端会同步更新 Xray inbound 用户。
- 吊销订阅：点击 `Revoke`，订阅失效并从 Xray 删除该用户。
- 查看流量：管理界面显示每个 token 的已用流量和配额进度。

## 重新部署

只重新生成配置并启动：

```bash
./setup.sh
docker compose up -d --build
```

重置运行配置但保留数据库：

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
curl -I http://127.0.0.1/
curl -I http://SERVER_ADDR/
docker exec nginx-sub tail -f /var/log/nginx/sub.access.log
```

如果 `xray` 容器启动失败，先确认：

- `.env` 已填写 `XRAY_PRIVATE_KEY`、`XRAY_PUBLIC_KEY`、`XRAY_SHORT_ID`、`XRAY_SERVER_NAME`、`XRAY_DEST`
- 已运行 `./setup.sh`
- `config/xray.json` 存在
- `XRAY_PORT` 没有被其他服务占用

## 文件说明

仓库只提交模板和源码，真实配置、密钥、数据库和日志不进 git。

| 文件 | 说明 |
|------|------|
| `.env.example` | 环境变量模板 |
| `.env` | 本地真实环境变量，不提交 |
| `config/config.example.json` | Xray 配置示例模板 |
| `config/nginx.example.conf` | nginx 配置模板 |
| `config/nginx.conf` | `setup.sh` 生成的 nginx 运行时配置，不提交 |
| `config/xray.json` | `setup.sh` 生成的 Xray 运行时配置，不提交 |
| `data/sub.db` | SQLite 数据库，不提交 |
| `web/front/` | Vue 3 前端管理界面 |
| `web/server/` | Go 管理服务源码 |
| `setup.sh` | 从 `.env` 生成运行时配置 |
| `cleanup.sh` | 清除生成的运行时配置 |

## 安全说明

- 每个用户有独立 UUID，互不影响。
- Xray gRPC API 只在 Docker 内网暴露，不要映射到公网。
- `XRAY_PRIVATE_KEY`、`ADMIN_SECRET` 和 `data/sub.db` 都不要提交或公开。
- 订阅链接含随机 token，泄露后应立即吊销并重新创建。
- 默认 nginx 只暴露 HTTP 管理和订阅；生产环境建议给管理面板和订阅配置 HTTPS。
