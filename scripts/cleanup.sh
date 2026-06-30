#!/bin/sh
# Remove all generated runtime files so setup.sh can start fresh.
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

rm -f config/nginx.conf
rm -f config/xray.json

if [ "$1" = "--with-db" ]; then
    rm -f data/sub.db
    echo "Removed database."
fi

echo "Cleaned. Run ./scripts/setup.sh to regenerate."
