#!/bin/sh
set -e

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: run as root, for example: sudo ./enable-bbr.sh"
    exit 1
fi

if [ "$(uname -s)" != "Linux" ]; then
    echo "Error: BBR can only be enabled on a Linux host"
    exit 1
fi

if ! grep -qw bbr /proc/sys/net/ipv4/tcp_available_congestion_control 2>/dev/null; then
    echo "Error: this kernel does not list bbr in tcp_available_congestion_control"
    echo "Available: $(cat /proc/sys/net/ipv4/tcp_available_congestion_control 2>/dev/null || echo unknown)"
    exit 1
fi

SYSCTL_FILE=/etc/sysctl.d/99-xray-bbr.conf

cat > "$SYSCTL_FILE" <<'EOF'
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
EOF

sysctl --system >/dev/null

echo "BBR status:"
sysctl net.core.default_qdisc
sysctl net.ipv4.tcp_congestion_control
sysctl net.ipv4.tcp_available_congestion_control
