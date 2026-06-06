#!/bin/sh
tz="${MARKPOST_TIMEZONE:-UTC}"
if [ ! -f "/usr/share/zoneinfo/$tz" ]; then
    echo "[cont-init.d] Warning: Invalid timezone '$tz', using UTC"
    tz="UTC"
fi
ln -snf "/usr/share/zoneinfo/$tz" /etc/localtime
echo "$tz" > /etc/timezone
echo "[cont-init.d] Timezone set to: $tz"

mkdir -p /app/data
echo "[cont-init.d] Data directory ready: /app/data"
