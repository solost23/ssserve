#!/bin/sh
set -e

if [ ! -f .env ]; then
    echo "Error: .env not found. Copy .env.example and fill in your values."
    exit 1
fi

. ./.env

if [ -z "$SS_DOMAIN" ] || [ "$SS_DOMAIN" = "example.com" ]; then
    echo "Error: SS_DOMAIN is not set in .env"
    exit 1
fi
if [ -z "$ADMIN_SECRET" ] || [ "$ADMIN_SECRET" = "replace-with-a-long-random-secret" ]; then
    echo "Error: ADMIN_SECRET is not set in .env"
    exit 1
fi

# nginx configs
sed "s|example.com|${SS_DOMAIN}|g" config/nginx.example.conf > config/nginx.conf
sed "s|example.com|${SS_DOMAIN}|g" config/nginx.bootstrap.example.conf > config/nginx.bootstrap.conf

# ssserver config (manager mode; placeholder server required for startup)
PORT_START="${SS_USER_PORT_START:-40200}"
PLACEHOLDER_PORT=$((PORT_START - 1))
cat > config/config.json << EOF
{
    "manager_address": "udp://0.0.0.0:6001",
    "server": "0.0.0.0",
    "server_port": ${PLACEHOLDER_PORT},
    "password": "placeholder-not-used",
    "method": "${SS_CIPHER:-aes-256-gcm}",
    "mode": "tcp_and_udp",
    "fast_open": true,
    "no_delay": true,
    "keep_alive": 15
}
EOF

mkdir -p data logs/nginx

echo ""
echo "Config generated successfully."
echo ""
echo "Next steps:"
echo "  1. Bootstrap cert:  NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx"
echo "  2. Get cert:        docker compose run --rm --entrypoint certbot certbot certonly --webroot -w /var/www/certbot -d ${SS_DOMAIN}"
echo "  3. Start all:       docker compose up -d"
echo ""
echo "Admin UI: https://${SS_DOMAIN}/"
echo "Login with ADMIN_SECRET from your .env"
