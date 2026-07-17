# syntax=docker/dockerfile:1.6

# ---- Build the Next.js dashboard ----
FROM node:22-alpine AS webbuild
WORKDIR /web
COPY dashboard/package.json dashboard/package-lock.json* ./
RUN npm ci --no-audit --no-fund
COPY dashboard/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build
# The `output: "export"` config writes the static site to /web/out

# ---- Build the Go binary with dashboard embedded ----
FROM golang:1.26-alpine AS gobuild
RUN apk add --no-cache build-base sqlite-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Pull the dashboard static export into the embed slot before `go build` runs.
RUN rm -rf internal/web/dist && mkdir -p internal/web/dist
COPY --from=webbuild /web/out/ /src/internal/web/dist/
ARG GIT_SHA=dev
ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w -X github.com/mecaca/waapi-gateway/internal/api.Version=${GIT_SHA}" -o /out/waapi-gateway ./cmd/server

# ---- Runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata sqlite-libs ffmpeg su-exec \
    && adduser -D -u 10001 gateway
WORKDIR /app
COPY --from=gobuild /out/waapi-gateway /app/waapi-gateway
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
EXPOSE 8080
VOLUME ["/app/storages"]
# Default to :8080 — matches Zeabur/Railway/Render/Fly/Cloud Run conventions.
# Override with HTTP_ADDR (or $PORT, which the binary also reads) for other setups.
ENV HTTP_ADDR=:8080 DB_DIALECT=sqlite3 DB_URI=file:/app/storages/gateway.db?_foreign_keys=on
# Entrypoint starts as root only to fix volume ownership, then drops to the
# unprivileged "gateway" user via su-exec before running the binary.
ENTRYPOINT ["/app/docker-entrypoint.sh"]
