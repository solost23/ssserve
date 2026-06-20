#!/bin/sh
# Remove all generated runtime files so setup.sh can start fresh.

rm -f config/nginx.conf
rm -f config/xray.json

if [ "$1" = "--with-db" ]; then
    rm -f data/sub.db
    echo "Removed database."
fi

echo "Cleaned. Run ./setup.sh to regenerate."
