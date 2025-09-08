# 魔搭-balance

一个用Go编写的魔搭API负载均衡服务，通过传入多个API Key实现负载均衡，只负责转发和负载均衡。

## 功能特性

- 🔑 支持多个API Key的负载均衡
- 🔄 轮询负载均衡策略
- 🏥 自动健康检查
- 📊 实时统计信息
- 🚀 高性能反向代理
- 📝 详细的日志记录
- ⚙️ 灵活的配置管理

## 快速开始

### 1. 配置API Keys

编辑 `config.json` 文件，添加你的魔搭API Keys：

```json
{
  "api_keys": [
    "your_api_key_1_here",
    "your_api_key_2_here",
    "your_api_key_3_here"
  ],
  "server_port": "8080",
  "target_url": "https://api-inference.modelscope.cn",
  "health_check": true,
  "health_path": "/v1/models"
}
```

### 2. 启动服务

使用Makefile：

```bash
# 构建并运行
make run

# 或者先构建再运行
make build
./modelscope-balance
```

使用启动脚本：

```bash
./start.sh
```

直接使用Go命令：

```bash
go run main.go
```

### 3. 使用服务

服务启动后，可以通过以下方式使用：

#### API调用示例

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-ai/DeepSeek-V3.1",
    "messages": [
      {"role": "system","content": "你是人工智能助手."},
      {"role": "user","content": "常见的十字花科植物有哪些？"}
    ]
  }'
```

#### 查看统计信息

```bash
curl http://localhost:8080/stats
```

#### 健康检查

```bash
curl http://localhost:8080/health
```

## 配置说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `api_keys` | array | [] | 魔搭API Key列表 |
| `server_port` | string | "8080" | 服务监听端口 |
| `target_url` | string | "https://api-inference.modelscope.cn" | 目标API服务器地址 |
| `health_check` | boolean | true | 是否启用健康检查 |
| `health_path` | string | "/v1/models" | 健康检查路径 |

## 负载均衡策略

当前实现采用轮询（Round Robin）策略：

1. 按顺序选择API Key
2. 跳过不健康的API Key
3. 自动重试直到找到可用的API Key
4. 如果所有API Key都不可用，返回503错误

## 健康检查

- 每30秒自动检查一次API Key的健康状态
- 通过访问 `/v1/models` 端点进行健康检查
- 如果API返回5xx错误或连接失败，标记为不健康
- 不健康的API Key会被跳过，直到恢复健康

## 监控和日志

### 统计信息

访问 `/stats` 端点获取详细的统计信息：

```json
{
  "api_key_1...": {
    "healthy": true,
    "last_used": "2024-01-01T12:00:00Z",
    "requests": 100
  },
  "api_key_2...": {
    "healthy": false,
    "last_used": "2024-01-01T11:30:00Z",
    "requests": 50
  }
}
```

### 日志输出

服务会输出详细的日志信息，包括：
- 请求转发信息
- API Key健康状态变化
- 错误和异常信息

## 项目结构

```
modelscope-balance/
├── main.go              # 主程序文件
├── config.json          # 配置文件
├── go.mod               # Go模块文件
├── Makefile             # 构建脚本
├── start.sh             # 启动脚本
├── README.md            # 说明文档
└── docs/
    └── 魔搭api调用.curl # API调用示例
```

## 开发和构建

### 开发环境要求

- Go 1.21+
- 魔搭API Key

### 构建命令

```bash
# 构建程序
make build

# 运行测试
make test

# 清理构建文件
make clean

# 安装依赖
make deps
```

### 自定义构建

```bash
# 构建为Linux可执行文件
GOOS=linux GOARCH=amd64 go build -o modelscope-balance-linux main.go

# 构建为Windows可执行文件
GOOS=windows GOARCH=amd64 go build -o modelscope-balance.exe main.go

# 构建为macOS可执行文件
GOOS=darwin GOARCH=amd64 go build -o modelscope-balance-darwin main.go
```

## 注意事项

1. **API Key安全**：请妥善保管你的API Key，不要将配置文件提交到版本控制系统
2. **负载均衡**：当前实现为简单的轮询策略，如需更复杂的策略可以自行扩展
3. **错误处理**：服务会自动处理API错误，但不包含重试逻辑
4. **性能考虑**：在高并发场景下，建议使用更专业的负载均衡解决方案

## 许可证

本项目采用MIT许可证，详见LICENSE文件。

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。