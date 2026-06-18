# ssserver

基于 Docker Compose 的 Shadowsocks 服务，含多用户 token 管理和 Clash 订阅。

## 架构

- **ssserver** — shadowsocks-rust，Manager 模式；每个用户独立端口和密码，动态添加/移除
- **subserver** — Go 服务，通过 Manager API 管理用户端口，生成动态 clash.yaml，每 5 分钟轮询流量统计
- **nginx** — 反代 subserver，提供前端管理界面

## 快速部署

```bash
# 1. 填写配置
cp .env.example .env
# 编辑 .env，填入 SS_DOMAIN、ADMIN_SECRET（其余可保持默认）

# 2. 生成配置文件
./setup.sh

# 3. 申请 TLS 证书（首次）
NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx
docker compose run --rm --entrypoint certbot certbot certonly --webroot -w /var/www/certbot -d <your-domain>

# 4. 启动所有服务
docker compose up -d
```

访问 `https://<your-domain>/`，输入 `ADMIN_SECRET` 登录管理界面。

## 管理操作

**创建订阅**：在管理界面填写名称，点击 Create，复制生成的订阅链接。

**吊销订阅**：点击 Revoke，立即失效。

**重置全部**：

```bash
./cleanup.sh && ./setup.sh
docker compose up -d
```

`--with-db` 参数同时删除数据库（所有 token 清空）：

```bash
./cleanup.sh --with-db && ./setup.sh
docker compose up -d
```

## 生成随机密钥

```bash
# ADMIN_SECRET 或其他随机值
openssl rand -hex 32
```

## 文件说明

仓库只提交模板文件，真实配置、密码、证书不进 git。

| 文件 | 说明 |
|------|------|
| `.env.example` | 配置变量模板 |
| `config/config.example.json` | ssserver 配置模板 |
| `config/nginx.example.conf` | nginx 主配置模板 |
| `config/nginx.bootstrap.example.conf` | 申请证书时用的临时 nginx 配置模板 |
| `server/` | Go 管理服务源码 |
| `web/index.html` | 前端管理界面 |
| `setup.sh` | 从 `.env` 生成所有运行时配置 |
| `cleanup.sh` | 清除生成的运行时配置 |

## 流量统计

subserver 每 5 分钟通过 ssserver Manager API 拉取每个用户端口的流量数据，写入 SQLite。管理界面实时显示用量和进度条。如果设置了 `quota_gb`，超额后订阅请求返回 403，Clash 无法更新配置。

流量计数在 ssserver 重启后会归零（重新从该次启动开始累计）。

## 安全说明

- 每个用户有独立 ss 密码，互不影响
- 订阅链接含随机 token，吊销后立即无法使用
- `ADMIN_SECRET` 是管理界面的唯一认证，请设置为强随机值
- 所有敏感文件（.env、config.json、证书）均在 .gitignore 中
