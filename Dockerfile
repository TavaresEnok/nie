# syntax=docker/dockerfile:1.7

# ---------- Frontend ----------
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- Backend ----------
FROM golang:1.27.3-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web /src/internal/webui/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nie ./cmd/nie

# ---------- Runtime ----------
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 -S nie \
    && adduser -u 10001 -S -D -H -G nie nie \
    && mkdir -p /data \
    && chown nie:nie /data
COPY --from=build /out/nie /usr/local/bin/nie
USER nie
WORKDIR /data
ENV NIE_ADDR=0.0.0.0:8080 \
    NIE_DATA_DIR=/data \
    NIE_WORKER=true \
    NIE_LOG_LEVEL=info
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=5 \
  CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/nie"]
