# ssserver

基于 Docker Compose 的 Shadowsocks 服务，包含多用户 token 管理、动态端口分配、流量统计和 Clash 订阅。

## 架构

- **ssserver**：shadowsocks-rust，Manager 模式；每个用户独立端口和密码。
- **subserver**：Go 管理服务；通过 Manager API 创建/吊销用户端口，生成 Clash YAML。
- **nginx**：反代管理界面和订阅接口，并处理 TLS 证书。
- **certbot**：申请和续期 Let's Encrypt 证书。

## 配置变量

复制示例配置：

```bash
cp .env.example .env
```

核心变量：

```env
SS_DOMAIN=example.com
SUB_BASE_URL=
SS_CIPHER=aes-256-gcm
SS_NAME=Tokyo
SS_USER_PORT_START=40200
ADMIN_SECRET=replace-with-a-long-random-secret
MANAGER_ADDR=ssserver:6001
NGINX_CONF=nginx.conf
```

- `SS_DOMAIN`：写入 Clash YAML 的 Shadowsocks 节点地址，也用于生成 nginx 证书配置。
- `SUB_BASE_URL`：管理界面返回的订阅入口。为空时默认使用 `https://SS_DOMAIN`。
- `ADMIN_SECRET`：管理界面登录密钥，请使用强随机值。
- `SS_USER_PORT_START`：用户端口起始值，默认发布 `40200-40299`。

生成随机密钥：

```bash
openssl rand -hex 32
```

## 普通域名部署

适合有稳定域名可直连访问的场景。

```bash
./setup.sh
NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx
docker compose run --rm --entrypoint certbot certbot certonly --webroot -w /var/www/certbot -d <your-domain>
docker compose down
docker compose up -d --build
```

访问：

```text
https://<your-domain>/
```

输入 `ADMIN_SECRET` 登录管理界面。

## Cloudflare Worker 订阅入口

适合手机或 Clash 无法稳定访问源站域名，但可以稳定访问 Cloudflare 的场景。Worker 只代理订阅请求；Shadowsocks 节点仍然直连 VPS。

`.env` 示例：

```env
SS_DOMAIN=38.47.105.9
SUB_BASE_URL=https://your-worker.your-name.workers.dev
```

含义：

- 管理界面生成的订阅链接使用 `SUB_BASE_URL`。
- Clash YAML 里的节点 `server` 使用 `SS_DOMAIN`，可以是 VPS IP。
- 源站 nginx 证书仍需要一个可用域名，例如 `tokyo-node.duckdns.org`，供 Worker 回源访问。

Worker 示例：

```js
const ORIGIN = "https://tokyo-node.duckdns.org";

export default {
  async fetch(request) {
    const url = new URL(request.url);

    if (!url.pathname.startsWith("/sub/")) {
      return new Response("not found", { status: 404 });
    }

    const upstream = new URL(url.pathname + url.search, ORIGIN);
    const resp = await fetch(upstream, {
      headers: {
        "User-Agent": request.headers.get("User-Agent") || "clash-sub-worker",
      },
    });

    const headers = new Headers(resp.headers);
    headers.set("content-type", "application/yaml; charset=utf-8");
    headers.set("cache-control", "no-store");
    headers.delete("content-disposition");

    return new Response(resp.body, {
      status: resp.status,
      headers,
    });
  },
};
```

部署 Worker 后更新 `.env`，再重启服务：

```bash
docker compose up -d --build subserver
```

已有 token 不需要重建；管理界面重新加载后会显示新的 Worker 订阅链接。

## 重新部署

保留数据库和 token，只重新生成配置并启动：

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

## 重新申请证书

```bash
docker compose down
rm -rf certbot/conf/live/<your-domain>
rm -rf certbot/conf/archive/<your-domain>
rm -f certbot/conf/renewal/<your-domain>.conf

NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx
docker compose run --rm --entrypoint certbot certbot certonly --webroot -w /var/www/certbot -d <your-domain>
docker compose down
docker compose up -d --build
```

## 管理操作

- 创建订阅：在管理界面填写名称，点击 `Create`，复制生成的订阅链接。
- 吊销订阅：点击 `Revoke`，订阅立即失效。
- 查看流量：管理界面会显示每个 token 的已用流量和配额进度。

## 验证

```bash
docker compose ps
docker exec nginx-sub nginx -T
curl -vkI https://127.0.0.1/
curl -vkI https://<your-domain>/
docker logs --tail 100 nginx-sub
```

查看订阅访问日志：

```bash
docker exec nginx-sub tail -f /var/log/nginx/sub.access.log
```

如果本机 `https://127.0.0.1/` 正常，但手机无法访问域名，优先排查手机 DNS、运营商链路、Clash 订阅更新是否走代理，或使用 Cloudflare Worker 订阅入口。

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
| `setup.sh` | 从 `.env` 生成运行时配置 |
| `cleanup.sh` | 清除生成的运行时配置 |

## 安全说明

- 每个用户有独立 ss 密码，互不影响。
- 订阅链接含随机 token，泄露后应立即吊销并重新创建。
- `ADMIN_SECRET` 是管理界面的唯一认证，请设置为强随机值。
- 订阅内容包含节点端口和密码，建议使用 HTTPS 或 Worker 入口，不建议长期使用 HTTP 明文订阅。
