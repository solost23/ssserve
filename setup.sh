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
if [ -z "$SS_PASSWORD" ] || [ "$SS_PASSWORD" = "replace-with-a-long-random-password" ]; then
    echo "Error: SS_PASSWORD is not set in .env"
    exit 1
fi

SS_PORT="${SS_PORT:-40105}"

# nginx configs
sed "s|example.com|${SS_DOMAIN}|g" config/nginx.example.conf > config/nginx.conf
sed "s|example.com|${SS_DOMAIN}|g" config/nginx.bootstrap.example.conf > config/nginx.bootstrap.conf

# ssserver config
sed "s|replace-with-a-long-random-password|${SS_PASSWORD}|g" config/config.example.json \
    | sed "s|40105|${SS_PORT}|g" > config/config.json

# subscribe
HASH=$(openssl rand -hex 16)
mkdir -p "config/subscribe/${HASH}"
sed \
    -e "s|example.com|${SS_DOMAIN}|g" \
    -e "s|40105|${SS_PORT}|g" \
    -e "s|replace-with-a-long-random-password|${SS_PASSWORD}|g" \
    config/subscribe/clash.example.yaml > "config/subscribe/${HASH}/clash.yaml"

echo ""
echo "Config generated successfully."
echo ""
echo "Subscription URL:"
echo "  https://${SS_DOMAIN}/sub/${HASH}/clash.yaml"
echo ""
echo "Next steps:"
echo "  1. Bootstrap cert:  NGINX_CONF=nginx.bootstrap.conf docker compose up -d nginx"
echo "  2. Get cert:        docker compose run --rm certbot certonly --webroot -w /var/www/certbot -d ${SS_DOMAIN}"
echo "  3. Start all:       docker compose up -d"
