# 无状态代理服务器测试示例

## 启动服务器

```bash
go run main.go
```

服务器将在 8080 端口启动，目标 URL 为 `https://api-inference.modelscope.cn`

## 基本测试

### 1. 健康检查

```bash
curl -s http://localhost:8080/health
```

### 2. 查看统计信息

```bash
curl -s http://localhost:8080/stats
```

### 3. 无 Authorization 头（返回错误）

```bash
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "deepseek-ai/DeepSeek-V3", "messages": [{"role": "user", "content": "Hello"}]}'
```

## API Key 测试

### 4. 单个 API Key

```bash
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key-here" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello"}]}'
```

### 5. 多个 API Keys（轮询测试）

```bash
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer key1,key2,key3" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello"}]}'
```

### 6. 测试失效机制（使用无效 key）

```bash
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid-key-1,invalid-key-2" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Hello"}]}'
```

## 高级测试

### 7. 连续请求测试轮询和失效恢复

```bash
# 第一次请求（使用第一个key）
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-key,invalid-key" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Test 1"}]}'

# 第二次请求（使用第二个key，会失效）
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-key,invalid-key" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Test 2"}]}'

# 第三次请求（应该只使用有效的key）
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-key,invalid-key" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Test 3"}]}'
```

### 8. 查看失效后的统计信息

```bash
curl -s http://localhost:8080/stats
```

### 9. 等待 10 秒后再次测试（失效 key 应该恢复）

```bash
sleep 10
curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-key,invalid-key" \
  -d '{"model": "qwen-turbo", "messages": [{"role": "user", "content": "Test after recovery"}]}'
```

## 使用说明

1. **Authorization 头格式**：必须是 `Bearer key1,key2,key3...`
2. **API keys 分隔**：用英文逗号分隔，不要有空格
3. **失效机制**：当 API key 返回 401 或 403 错误时，会被标记为失效 10 秒
4. **失效恢复**：失效期间该 key 不会被使用，10 秒后自动恢复
5. **轮询机制**：服务器会轮询使用可用的 API keys

## 一键测试脚本

运行完整的测试脚本：

```bash
./test_curl_commands.sh
```

或者运行自动化测试：

```bash
./test_new_implementation.sh
```
