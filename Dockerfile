# syntax=docker/dockerfile:1.6

# ---- Build the Go binary ----
FROM golang:1.26-alpine AS gobuild
RUN apk add --no-cache build-base sqlite-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w" -o /out/waapi-gateway ./cmd/server

# ---- Build the Next.js dashboard ----
FROM node:22-alpine AS webbuild
WORKDIR /web
COPY dashboard/package.json dashboard/package-lock.json* ./
RUN npm ci --no-audit --no-fund
COPY dashboard/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- Runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata sqlite-libs ffmpeg \
    && adduser -D -u 10001 gateway
WORKDIR /app
COPY --from=gobuild /out/waapi-gateway /app/waapi-gateway
# Optional: copy the dashboard standalone build if you serve it from the same container.
# For the default setup the dashboard runs in its own container (see docker-compose).
USER gateway
EXPOSE 3000
VOLUME ["/app/storages"]
ENV HTTP_ADDR=:3000 DB_DIALECT=sqlite3 DB_URI=file:/app/storages/gateway.db?_foreign_keys=on
ENTRYPOINT ["/app/waapi-gateway"]
