FROM oven/bun:1 AS web-builder
WORKDIR /src/web

COPY web/package.json ./
COPY web/bun.lock* ./
RUN if [ -f bun.lock ] || [ -f bun.lockb ]; then bun install --frozen-lockfile; else bun install; fi

COPY web/. .
RUN bun run build

FROM golang:1.26-alpine AS backend-builder
WORKDIR /src/backend

COPY backend/go.mod ./
COPY backend/go.sum ./
RUN go mod download

COPY backend/. .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
  bash \
  ca-certificates \
  postgresql-client \
  python3 \
  python3-venv \
  python3-pip \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=web-builder /src/web/dist /app/web-dist
COPY --from=backend-builder /out/api /app/api
COPY --from=backend-builder /out/worker /app/worker
COPY scripts/python /app/scripts/python
COPY docker/start-all-in-one.sh /app/start.sh

RUN chmod +x /app/api /app/worker /app/start.sh

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
