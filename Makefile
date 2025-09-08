.PHONY: build run clean test help

# 默认目标
all: build

# 构建程序
build:
	go build -o modelscope-balance main.go

# 运行程序
run:
	go run main.go

# 清理构建文件
clean:
	rm -f modelscope-balance

# 运行测试
test:
	go test -v ./...

# 显示帮助信息
help:
	@echo "可用的命令："
	@echo "  build     - 构建程序"
	@echo "  run       - 运行程序"
	@echo "  clean     - 清理构建文件"
	@echo "  test      - 运行测试"
	@echo "  help      - 显示此帮助信息"

# 安装依赖
deps:
	go mod tidy