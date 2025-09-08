#!/bin/bash

# 魔搭API负载均衡服务测试脚本

echo "开始测试魔搭API负载均衡服务..."

# 检查服务是否运行
if ! pgrep -f "modelscope-balance" > /dev/null; then
    echo "警告: modelscope-balance 服务未运行"
    echo "请先启动服务: ./start.sh"
    exit 1
fi

# 测试健康检查端点
echo "1. 测试健康检查端点..."
health_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$health_response" -eq 200 ]; then
    echo "✅ 健康检查端点正常"
else
    echo "❌ 健康检查端点异常，状态码: $health_response"
fi

# 测试统计信息端点
echo "2. 测试统计信息端点..."
stats_response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/stats)
if [ "$stats_response" -eq 200 ]; then
    echo "✅ 统计信息端点正常"
    echo "统计信息:"
    curl -s http://localhost:8080/stats | jq .
else
    echo "❌ 统计信息端点异常，状态码: $stats_response"
fi

# 测试API转发（需要配置有效的API Key）
echo "3. 测试API转发..."
api_response=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-ai/DeepSeek-V3.1",
    "messages": [
      {"role": "system","content": "你是人工智能助手."},
      {"role": "user","content": "测试消息"}
    ]
  }')

if [ "$api_response" -eq 200 ] || [ "$api_response" -eq 401 ] || [ "$api_response" -eq 429 ]; then
    echo "✅ API转发正常，状态码: $api_response"
    if [ "$api_response" -eq 401 ]; then
        echo "   (401错误是正常的，表示需要配置有效的API Key)"
    fi
    if [ "$api_response" -eq 429 ]; then
        echo "   (429错误是正常的，表示API Key可能达到限制)"
    fi
else
    echo "❌ API转发异常，状态码: $api_response"
fi

echo "测试完成！"