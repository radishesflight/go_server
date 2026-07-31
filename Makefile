# =====================================================
# go_server Makefile
# 常用命令:make help 查看所有命令
# =====================================================

# ---------- 变量 ----------
APP        := go_server
BIN_DIR    := bin
CMD_DIR    := cmd
MAIN_PKG   := ./cmd/server
CONFIG     := config/config.yaml
GO         := go
GOFLAGS    := -v
LDFLAGS    := -s -w

# 默认目标
.DEFAULT_GOAL := help

# 防止 make 把这些当作文件
.PHONY: help build run test tidy clean fmt vet lint tidy-deps

# =====================================================
# 帮助
# =====================================================
help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# =====================================================
# 构建
# =====================================================
build: ## 编译主程序到 bin/
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP).exe $(MAIN_PKG)
	@echo "✓ 构建完成: $(BIN_DIR)/$(APP).exe"

# =====================================================
# 运行
# =====================================================
run: ## 运行主程序(本地开发用)
	$(GO) run $(MAIN_PKG)

# =====================================================
# 测试
# =====================================================
test: ## 运行所有测试
	$(GO) test ./... -v

test-coverage: ## 运行测试并生成覆盖率报告
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ 覆盖率报告: coverage.html"

# =====================================================
# 依赖治理
# =====================================================
tidy: ## go mod tidy
	$(GO) mod tidy
	@echo "✓ go.mod / go.sum 已整理"

tidy-deps: ## 查看依赖图
	$(GO) mod graph

# =====================================================
# 代码质量
# =====================================================
fmt: ## 格式化代码
	$(GO) fmt ./...
	@echo "✓ 格式化完成"

vet: ## 静态检查
	$(GO) vet ./...
	@echo "✓ vet 完成"

# =====================================================
# 清理
# =====================================================
clean: ## 清理构建产物
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@rm -f $(APP) $(APP).exe
	@echo "✓ 清理完成"
