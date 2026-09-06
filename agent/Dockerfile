# Agent 构建
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY agent/ ./agent
WORKDIR /app/agent
RUN CGO_ENABLED=0 go build -o msmp-agent .

# 运行镜像
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/agent/msmp-agent .
ENTRYPOINT ["./msmp-agent"]