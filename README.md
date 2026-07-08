# xray-sub

基于 Docker Compose 的 Xray VLESS REALITY 服务，包含用户 token 管理、动态 UUID 下发、流量统计、配额控制和 Clash Meta 订阅。

## 架构

- **xray**：VLESS + REALITY + Vision，公网只暴露一个 TCP 入口。
- **subserver**：Go 管理服务，通过 Xray gRPC API 创建、暂停、恢复和删除用户 UUID。
- **nginx**：提供管理界面、API 反向代理和订阅接口。
- **web/front**：Vue 3 管理界面。

## 一键部署

在新 VPS 上 clone 仓库后，直接运行：

```bash
./deploy.sh --node Singapore --addr <server-ip>.sslip.io --enable-bbr
```

常用参数：

```bash
./deploy.sh --node Tokyo --addr 202.182.111.110.sslip.io
./deploy.sh --node Singapore --addr <server-ip>.sslip.io --force
./deploy.sh --node Singapore --addr <server-ip>.sslip.io --port 443 --enable-bbr
```

`deploy.sh` 会自动：

- 检查 Docker / Docker Compose；Linux 主机缺少 Docker 时会尝试安装。
- 生成 `.env`、`ADMIN_SECRET`、REALITY 密钥和 `XRAY_SHORT_ID`。
- 使用固定的 Xray 镜像 digest，避免 `latest` 或 tag 漂移。
- 生成 `config/nginx.conf` 和 `config/xray.json`。
- 执行 `docker compose up -d --build`。
- 输出管理界面地址和节点地址。

如果 `.env` 已存在，默认会保留现有节点配置，只重新生成运行时配置并重启容器。需要重建节点配置时加 `--force`。

## 手动部署

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
SERVER_ADDR=202.182.111.110.sslip.io
NODE_NAME=Tokyo
ADMIN_SECRET=replace-with-a-long-random-secret
XRAY_PORT=443
XRAY_IMAGE=ghcr.io/xtls/xray-core@sha256:592ec4d11f656db95598d01e76dbcc6e002d67360b96a5436500a938230f52c7
XRAY_PRIVATE_KEY=replace-with-reality-private-key
XRAY_PUBLIC_KEY=replace-with-reality-public-key
XRAY_SHORT_ID=replacehex
XRAY_SERVER_NAME=www.cloudflare.com
XRAY_DEST=www.cloudflare.com:443
XRAY_SYNC_INTERVAL=5m
XRAY_FULL_SYNC_INTERVAL=10m
```

`SERVER_ADDR` 会用于：

- 管理界面地址：`http://SERVER_ADDR/`
- 订阅链接：`http://SERVER_ADDR/sub/<token>/clash.yaml`
- 客户端节点里的 `server`

`SERVER_ADDR` 会写入订阅链接和客户端节点里的 `server` 字段。客户端如果通过域名连接就填域名；如果直接通过 IP 连接就填公网 IP。`XRAY_PORT` 是 VLESS REALITY 的公网端口，默认使用 443。

`XRAY_SYNC_INTERVAL` 控制 subserver 后台检查 Xray 用户注册状态的频率，默认 `5m`；`XRAY_FULL_SYNC_INTERVAL` 控制强制全量提交 active token 的频率，默认 `10m`。创建、恢复 token 仍会立即同步到 Xray。

## 可选网络优化

如果服务器主要用于跨境 TCP 代理，可以在宿主机开启 BBR：

```bash
sudo ./scripts/enable-bbr.sh
```

脚本会写入 `/etc/sysctl.d/99-xray-bbr.conf`，设置：

```conf
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
```

这是宿主机级别设置，不在 Docker 容器内生效。开启后可用脚本输出的 `sysctl` 结果确认当前拥塞控制算法是否为 `bbr`。

## 生成配置

运行：

```bash
./scripts/setup.sh
```

它会根据 `.env` 生成运行时配置：

- `config/nginx.conf`
- `config/xray.json`

`config/xray.json` 是 Docker Compose 挂载给 Xray 的实际配置文件：

```yaml
./config/xray.json:/etc/xray/config.json:ro
```

这个文件包含 REALITY 私钥，所以不会提交到 git。新机器部署时，本地没有 `config/xray.json` 是正常的，先运行 `./deploy.sh` 或 `./scripts/setup.sh` 生成它。

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
- ClashX Meta / Clash Lite：使用 `Clash` 订阅链接。
- v2rayN / v2rayNG 等客户端：使用 `VLESS` 链接。
- 暂停或恢复用户：在管理界面操作，后端会同步更新 Xray inbound 用户。
- 吊销订阅：点击 `Revoke`，订阅失效并从 Xray 删除该用户。
- 查看流量：管理界面显示每个 token 的已用流量和配额进度。

## 重新部署

使用现有 `.env` 一键重新部署：

```bash
./deploy.sh
```

只重新生成配置并启动：

```bash
./scripts/setup.sh
docker compose up -d --build
```

重置运行配置但保留数据库：

```bash
./scripts/cleanup.sh
./scripts/setup.sh
docker compose up -d --build
```

完全重置，包括删除所有 token 和统计数据：

```bash
./scripts/cleanup.sh --with-db
./scripts/setup.sh
docker compose up -d --build
```

## 验证

```bash
curl -I http://127.0.0.1/
curl -I http://SERVER_ADDR/
docker exec nginx-sub tail -f /var/log/nginx/sub.access.log
```

如果 `xray` 容器启动失败，先确认：

- `.env` 已填写 `XRAY_IMAGE`、`XRAY_PRIVATE_KEY`、`XRAY_PUBLIC_KEY`、`XRAY_SHORT_ID`、`XRAY_SERVER_NAME`、`XRAY_DEST`
- 已运行 `./deploy.sh` 或 `./scripts/setup.sh`
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
| `config/nginx.conf` | `scripts/setup.sh` 生成的 nginx 运行时配置，不提交 |
| `config/xray.json` | `scripts/setup.sh` 生成的 Xray 运行时配置，不提交 |
| `data/sub.db` | SQLite 数据库，不提交 |
| `web/front/` | Vue 3 前端管理界面 |
| `web/server/` | Go 管理服务源码 |
| `deploy.sh` | 仓库内一键部署入口 |
| `scripts/setup.sh` | 从 `.env` 生成运行时配置 |
| `scripts/cleanup.sh` | 清除生成的运行时配置 |
| `scripts/enable-bbr.sh` | 开启宿主机 BBR 和 TCP 参数优化 |

## 安全说明

- 每个用户有独立 UUID，互不影响。
- Xray gRPC API 只在 Docker 内网暴露，不要映射到公网。
- `XRAY_PRIVATE_KEY`、`ADMIN_SECRET` 和 `data/sub.db` 都不要提交或公开。
- 订阅链接含随机 token，泄露后应立即吊销并重新创建。
- 默认 nginx 只暴露 HTTP 管理和订阅；生产环境建议给管理面板和订阅配置 HTTPS。
