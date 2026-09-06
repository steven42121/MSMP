# 后端构建
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY server/ ./server
WORKDIR /app/server
RUN CGO_ENABLED=0 go build -o msmp-server .

# 运行镜像
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server/msmp-server .
COPY server/config.yaml .
VOLUME ["/app/data"]
EXPOSE 8080
ENTRYPOINT ["./msmp-server"]