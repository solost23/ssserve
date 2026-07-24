#!/bin/sh
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

if [ ! -f .env ]; then
    echo "Error: .env not found. Run ./deploy.sh or copy .env.example and fill in your values."
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
if [ -z "$XRAY_PRIVATE_KEY" ] || [ -z "$XRAY_PUBLIC_KEY" ]; then
    echo "Error: XRAY_PRIVATE_KEY and XRAY_PUBLIC_KEY are required. Generate them with: docker run --rm ghcr.io/xtls/xray-core:26.7.11 x25519"
    exit 1
fi
if [ -z "$XRAY_SHORT_ID" ] || [ -z "$XRAY_SERVER_NAME" ] || [ -z "$XRAY_DEST" ]; then
    echo "Error: XRAY_SHORT_ID, XRAY_SERVER_NAME and XRAY_DEST are required in .env"
    exit 1
fi

XRAY_PORT=${XRAY_PORT:-443}
XRAY_INBOUND_TAG=${XRAY_INBOUND_TAG:-vless-in}
NODE_NAME=${NODE_NAME:-Tokyo}
XRAY_TCP_CONGESTION=${XRAY_TCP_CONGESTION:-}

XRAY_TCP_CONGESTION_FIELD=
if [ -n "$XRAY_TCP_CONGESTION" ]; then
    XRAY_TCP_CONGESTION_FIELD=$(printf ',\n          "tcpCongestion": "%s"' "$XRAY_TCP_CONGESTION")
fi

cp config/nginx.example.conf config/nginx.conf

cat > config/xray.json << EOF
{
  "log": {
    "access": "/dev/stdout",
    "error": "/dev/stderr",
    "loglevel": "debug"
  },
  "api": {
    "tag": "api",
    "services": [
      "HandlerService",
      "StatsService"
    ]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true,
        "bufferSize": 512,
        "connIdle": 300,
        "uplinkOnly": 2,
        "downlinkOnly": 5
      }
    }
  },
  "inbounds": [
    {
      "tag": "${XRAY_INBOUND_TAG}",
      "listen": "0.0.0.0",
      "port": ${XRAY_PORT},
      "protocol": "vless",
      "settings": {
        "clients": [],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "sockopt": {
          "tcpFastOpen": true${XRAY_TCP_CONGESTION_FIELD}
        },
        "realitySettings": {
          "show": true,
          "dest": "${XRAY_DEST}",
          "xver": 0,
          "serverNames": [
            "${XRAY_SERVER_NAME}"
          ],
          "privateKey": "${XRAY_PRIVATE_KEY}",
          "shortIds": [
            "${XRAY_SHORT_ID}"
          ]
        }
      }
    },
    {
      "tag": "api",
      "listen": "0.0.0.0",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      }
    }
  ],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom",
      "streamSettings": {
        "sockopt": {
          "tcpFastOpen": true${XRAY_TCP_CONGESTION_FIELD}
        }
      }
    },
    {
      "tag": "blocked",
      "protocol": "blackhole"
    }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "inboundTag": [
          "api"
        ],
        "outboundTag": "api"
      }
    ]
  }
}
EOF

mkdir -p data logs/nginx
chmod 644 config/nginx.conf config/xray.json

echo ""
echo "Config generated successfully."
echo ""
echo "Next steps:"
echo "  1. Start all:       docker compose up -d --build"
echo ""
echo "Admin UI: http://${SERVER_ADDR}/"
echo "Proxy:    ${SERVER_ADDR}:${XRAY_PORT} (${NODE_NAME})"
echo "Login with ADMIN_SECRET from your .env"
