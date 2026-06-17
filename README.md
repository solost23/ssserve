# ssserve

这是一个基于 Docker Compose 的 Shadowsocks 服务部署仓库，同时用 nginx
提供 Clash 订阅文件下载。

## 文件说明

仓库只提交模板文件。真实运行配置、密码、订阅 token 和证书私钥不进入 git。

- `.env.example`：环境变量模板。
- `config/config.example.json`：Shadowsocks 服务端配置模板。
- `config/nginx.bootstrap.example.conf`：首次申请证书时使用的 HTTP-only nginx 配置模板。
- `config/nginx.example.conf`：启用 HTTPS 后使用的 nginx 配置模板。
- `config/subscribe/clash.example.yaml`：Clash 订阅配置模板，按用户复制到 `config/subscribe/<hash>/clash.yaml`。

以下本地运行文件会被忽略，不会提交：

- `.env`
- `config/config.json`
- `config/nginx.conf`
- `config/nginx.bootstrap.conf`
- `config/subscribe/**/*.yaml`
- `certbot/conf/`
- `logs/`

## 初始化配置

先复制模板文件，再替换里面的占位符：

```bash
cp .env.example .env
cp config/config.example.json config/config.json
cp config/nginx.bootstrap.example.conf config/nginx.bootstrap.conf
cp config/nginx.example.conf config/nginx.conf

# 为每个用户生成一个随机 hash，创建对应的订阅目录
HASH=$(openssl rand -hex 16)
mkdir -p config/subscribe/$HASH
cp config/subscribe/clash.example.yaml config/subscribe/$HASH/clash.yaml
```

需要替换的内容：

- `example.com`：你的公网域名。
- `40105`：Shadowsocks TCP/UDP 端口；如果不改端口，可以保持默认值。
- `replace-with-a-long-random-password`：Shadowsocks 密码，在 `config.json` 和对应用户的 `clash.yaml` 里保持一致。

订阅地址为 `https://<域名>/sub/<hash>/clash.yaml`，hash 即目录名。每个用户独立 hash，互不影响，吊销时删除对应目录即可。

部署前建议生成随机密码：

```bash
openssl rand -base64 32
```

## 首次申请证书

首次申请证书时，先用 bootstrap 配置启动 nginx：

```bash
NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx
```

证书生成到 `certbot/conf/` 后，切回 HTTPS 配置并启动完整服务：

```bash
docker compose up -d
```

## 安全注意事项

如果真实密码、订阅 token 或证书私钥曾经提交到 git 历史中，需要轮换它们。
从当前提交中删除文件并不会清除已有 git 历史里的敏感内容。
