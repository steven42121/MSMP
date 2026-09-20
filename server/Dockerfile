# ── 构建阶段 ──
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG BUILD_TIME
ARG COMMIT
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.BuildTime=${BUILD_TIME} \
    -X main.Commit=${COMMIT}" \
    -o /msmp-server .

# ── 运行阶段 ──
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /msmp-server .
COPY config.yaml ./config.yaml
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1
ENTRYPOINT ["./msmp-server"]