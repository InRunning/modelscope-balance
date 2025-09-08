#!/bin/bash

# 魔搭API负载均衡服务启动脚本

echo "正在启动魔搭API负载均衡服务..."

# 检查配置文件是否存在
if [ ! -f "config.json" ]; then
    echo "错误: config.json 文件不存在，请先创建配置文件"
    exit 1
fi

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: Go环境未安装，请先安装Go"
    exit 1
fi

# 构建程序
echo "正在构建程序..."
go build -o modelscope-balance main.go

if [ $? -ne 0 ]; then
    echo "构建失败"
    exit 1
fi

echo "构建成功，正在启动服务..."

# 启动服务
./modelscope-balance