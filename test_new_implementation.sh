#!/bin/bash

# 测试新的无状态代理服务器实现

echo "启动服务器..."
# 在后台启动服务器
./modelscope-balance &
SERVER_PID=$!

# 等待服务器启动
sleep 2

echo "测试服务器健康状态..."
# 测试健康检查端点
curl -s http://localhost:8080/health

echo -e "\n\n测试统计信息端点..."
# 测试统计信息端点
curl -s http://localhost:8080/stats

echo -e "\n\n测试代理功能（无Authorization头）..."
# 测试无Authorization头的情况
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello"}]}'

echo -e "\n\n测试代理功能（有Authorization头）..."
# 测试有Authorization头的情况（使用假的API keys）
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer fake-key1,fake-key2,fake-key3" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello"}]}'

echo -e "\n\n停止服务器..."
# 停止服务器
kill $SERVER_PID

echo "测试完成！"