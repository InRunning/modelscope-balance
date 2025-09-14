.PHONY: build run clean test help deploy

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
	@echo "  deploy    - 部署程序（停止旧进程，重新构建并启动）"
	@echo "  help      - 显示此帮助信息"

# 安装依赖
deps:
	go mod tidy

# 部署程序（停止旧进程，重新构建并启动）
deploy:
	@echo "停止旧进程..."
	@ps aux | grep "modelscope-balance" | grep -v grep | awk '{print $$2}' | xargs -r kill -9 || true
	@sleep 1
	@echo "重新构建程序..."
	@make build
	@echo "启动新进程..."
	@nohup ./modelscope-balance > /home/ubuntu/nohup.out 2>&1 &
	@echo "部署完成，进程已在后台运行"