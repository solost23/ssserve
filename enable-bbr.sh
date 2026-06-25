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

# socket buffer — allow kernel to auto-tune up to 64 MB
net.core.rmem_max=67108864
net.core.wmem_max=67108864
net.ipv4.tcp_rmem=4096 87380 67108864
net.ipv4.tcp_wmem=4096 65536 67108864

# connection backlog
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535

# ephemeral port range
net.ipv4.ip_local_port_range=1024 65535

# TCP Fast Open (client + server)
net.ipv4.tcp_fastopen=3

# TIME_WAIT reuse and faster cleanup
net.ipv4.tcp_tw_reuse=1
net.ipv4.tcp_fin_timeout=30

# TCP keepalive
net.ipv4.tcp_keepalive_time=300
net.ipv4.tcp_keepalive_intvl=15
net.ipv4.tcp_keepalive_probes=5
EOF

sysctl --system >/dev/null

echo "BBR status:"
sysctl net.core.default_qdisc
sysctl net.ipv4.tcp_congestion_control
sysctl net.ipv4.tcp_available_congestion_control
