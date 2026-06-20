FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -ldflags="-s -w" -o subserver .

FROM ghcr.io/xtls/xray-core:latest AS xray

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=xray /usr/local/bin/xray /usr/local/bin/xray
COPY --from=builder /app/subserver .
CMD ["./subserver"]
