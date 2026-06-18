#!/bin/sh
set -e

if [ ! -f .env ]; then
    echo "Error: .env not found. Copy .env.example and fill in your values."
    exit 1
fi

. ./.env

if [ -z "$SERVER_ADDR" ] || [ "$SERVER_ADDR" = "127.0.0.1" ]; then
    echo "Error: SERVER_ADDR is not set in .env"
    exit 1
fi
if [ -z "$ADMIN_SECRET" ] || [ "$ADMIN_SECRET" = "replace-with-a-long-random-secret" ]; then
    echo "Error: ADMIN_SECRET is not set in .env"
    exit 1
fi

# nginx configs
cp config/nginx.example.conf config/nginx.conf

# ssmanager config
cat > config/config.json << EOF
{
    "manager_address": "0.0.0.0:6001",
    "server": "0.0.0.0",
    "server_port": 40199,
    "password": "placeholder",
    "method": "${SS_CIPHER:-chacha20-ietf-poly1305}",
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
echo "  1. Start all:       docker compose up -d --build"
echo ""
echo "Admin UI: http://${SERVER_ADDR}/"
echo "Login with ADMIN_SECRET from your .env"
