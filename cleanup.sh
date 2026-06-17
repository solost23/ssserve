#!/bin/sh
# Remove all generated runtime files so setup.sh can start fresh.

rm -f config/nginx.conf
rm -f config/nginx.bootstrap.conf
rm -f config/config.json

# Remove subscribe dirs (keep example files)
find config/subscribe -mindepth 1 -maxdepth 1 -type d | xargs rm -rf

echo "Cleaned. Run ./setup.sh to regenerate."
