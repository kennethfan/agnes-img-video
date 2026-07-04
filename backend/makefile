# 用法: make [build|run|clean|all]

# ---------- 变量配置 ----------
BINARY_NAME = server
BINARY_DIR = bin
MAIN_PATH = ./cmd/server

# Go 相关
GO = go
GOFLAGS = -mod=mod
LDFLAGS = -s -w

# 颜色输出（可选）
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all build run clean test help

# ---------- 默认目标 ----------
all: build

# ---------- 构建 ----------
build:
	@echo "$(GREEN)▶ 正在构建 $(BINARY_NAME)...$(RESET)"
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✅ 构建完成: $(BINARY_DIR)/$(BINARY_NAME)$(RESET)"

# ---------- 运行（构建后自动运行） ----------
run: build
	@echo "$(GREEN)▶ 启动服务...$(RESET)"
	$(BINARY_DIR)/$(BINARY_NAME)

# ---------- 清理 ----------
clean:
	@echo "$(YELLOW)▶ 清理构建产物...$(RESET)"
	@rm -rf $(BINARY_DIR)
	@echo "$(GREEN)✅ 清理完成$(RESET)"

# ---------- 测试 ----------
test:
	@echo "$(GREEN)▶ 运行测试...$(RESET)"
	$(GO) test ./... -v

# ---------- 帮助 ----------
help:
	@echo "可用目标："
	@echo "  build   - 编译 Go 服务器到 $(BINARY_DIR)/$(BINARY_NAME)"
	@echo "  run     - 编译并运行服务器"
	@echo "  clean   - 删除 $(BINARY_DIR) 目录"
	@echo "  test    - 运行所有测试"
	@echo "  all     - 默认目标，等价于 build"
	@echo "  help    - 显示此帮助"
