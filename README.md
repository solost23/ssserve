# ssserver

基于 Docker Compose 的 Shadowsocks 服务，nginx 提供 Clash 订阅。

## 快速部署

```bash
# 1. 填写配置
cp .env.example .env
# 编辑 .env，填入 SS_DOMAIN、SS_PORT、SS_PASSWORD

# 2. 生成配置文件（会输出订阅链接）
./setup.sh

# 3. 申请 TLS 证书（首次）
NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx
docker compose run --rm --entrypoint certbot certbot certonly --webroot -w /var/www/certbot -d <your-domain>

# 4. 启动所有服务
docker compose up -d
```

## 轮换密码或订阅链接

重新编辑 `.env`，再跑一次 `./setup.sh` 并重启服务即可。旧的订阅目录需手动删除 `config/subscribe/`。

## 文件说明

仓库只提交模板文件，真实配置、密码、证书不进 git。

- `.env.example` — 配置变量模板
- `config/config.example.json` — ssserver 配置模板
- `config/nginx.example.conf` — nginx 主配置模板
- `config/nginx.bootstrap.example.conf` — 申请证书时用的临时 nginx 配置模板
- `config/subscribe/clash.example.yaml` — Clash 订阅模板
- `setup.sh` — 从 `.env` 生成所有运行时配置

## 安全注意事项

如果密码或订阅链接曾经泄露，更新 `.env` 里的密码，重跑 `./setup.sh` 并重启服务即可轮换。旧的订阅目录记得手动删除。
