# =====================================================
# 多阶段构建
# 阶段 1:builder  - 编译二进制
# 阶段 2:runtime  - 运行时最小镜像
# =====================================================

# 阶段 1:builder
FROM golang:1.26.5-alpine AS builder

# 国内构建
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -tags 'netgo osusergo' -ldflags "-s -w -extldflags '-static'" -o /out/go_server ./cmd/server

# 阶段 2:runtime
FROM alpine:3.19

# 国内构建
RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories

# CA + 时区 + curl
RUN apk add --no-cache ca-certificates tzdata curl

ENV TZ=Asia/Shanghai
WORKDIR /app

# 拷二进制
COPY --from=builder /out/go_server /app/go_server

# 配置模板(运行时由 ENV 或 -v 注入真实 config)
COPY config/config.example.yaml /app/config/config.yaml

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS http://localhost:8080/api/login -X POST -H "Content-Type: application/json" -d '{}' || exit 1

ENTRYPOINT ["/app/go_server"]