# =====================================================
# 多阶段构建
# 阶段 1:builder  - 编译二进制
# 阶段 2:runtime  - 运行时最小镜像
# =====================================================

# 阶段 1:builder
FROM golang:1.25-alpine AS builder

# 国内构建可取消下行注释(Alpine 镜像源)
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

WORKDIR /src

# 单独 copy go.mod / go.sum,利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# copy 源码
COPY . .

# 编译 server 主程序(交叉编译 CGO 关闭,产出静态二进制)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/go_server ./cmd/server

# 可选:也编译 fix 工具
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/fix ./cmd/fix || true

# 阶段 2:runtime
FROM alpine:3.19

# 国内构建可取消下行注释
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装 CA 证书 + 时区数据 + curl(健康检查用)
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区(可改)
ENV TZ=Asia/Shanghai

# 工作目录
WORKDIR /app

# 从 builder 拷二进制
COPY --from=builder /out/go_server /app/go_server
COPY --from=builder /out/fix /app/fix

# 配置目录(容器内运行时通过 -v 挂载真实 config.yaml,或 ENV 注入)
COPY config/config.example.yaml /app/config/config.example.yaml

# 暴露端口(实际端口由 SERVER_PORT 环境变量决定)
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS http://localhost:8080/api/login -X POST -H "Content-Type: application/json" -d '{}' || exit 1

# 启动
ENTRYPOINT ["/app/go_server"]
