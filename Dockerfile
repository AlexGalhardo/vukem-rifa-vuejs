# syntax=docker/dockerfile:1

# 1) Frontend assets (Vite + Vue + Tailwind) with Bun.
FROM oven/bun:1.4.3-alpine AS assets
WORKDIR /src
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --ignore-scripts
COPY tsconfig.json vite.config.ts ./
COPY resources ./resources
RUN bun run build

# 2) Go binary with views, translations and the Vite build embedded.
FROM golang:1.27.2-alpine3.24 AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=assets /src/resources/public/build ./resources/public/build
RUN go build -trimpath -ldflags "-s -w" -o /out/rifa ./cmd/app

# 3) Runtime: no resources/ directory on disk, so the embedded copies are served.
FROM alpine:3.24.2
RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 rifa
WORKDIR /app
COPY --from=build /out/rifa /app/rifa
USER rifa
ENV APP_ENV=production HOST=0.0.0.0 PORT=3000
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
	CMD wget -qO- "http://127.0.0.1:${PORT}/up" >/dev/null || exit 1
# Migrations are additive and idempotent: run them before serving.
CMD ["sh", "-c", "/app/rifa migrate && exec /app/rifa"]
