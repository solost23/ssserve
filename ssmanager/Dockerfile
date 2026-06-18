FROM alpine:latest AS downloader
ARG SS_VERSION=1.24.0
ARG TARGETARCH
RUN apk add --no-cache wget xz && \
    case "$TARGETARCH" in \
      arm64) ARCH=aarch64-unknown-linux-musl ;; \
      *) ARCH=x86_64-unknown-linux-musl ;; \
    esac && \
    wget -O /tmp/ss.tar.xz \
      "https://github.com/shadowsocks/shadowsocks-rust/releases/download/v${SS_VERSION}/shadowsocks-v${SS_VERSION}.${ARCH}.tar.xz" && \
    tar -xJf /tmp/ss.tar.xz -C /usr/local/bin ssmanager && \
    rm /tmp/ss.tar.xz && \
    chmod +x /usr/local/bin/ssmanager

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=downloader /usr/local/bin/ssmanager /usr/local/bin/ssmanager
ENTRYPOINT ["ssmanager"]
CMD ["-c", "/etc/shadowsocks-rust/config.json", "--manager-address", "0.0.0.0:6001"]
