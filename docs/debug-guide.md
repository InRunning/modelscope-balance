# VSCode 调试指南

本文档介绍如何使用 VSCode 调试魔搭 API 负载均衡服务。

## 前置条件

1. 安装 VSCode
2. 安装 Go 扩展包
3. 安装 Go 工具链

## 调试配置

### launch.json 配置说明

项目已配置了多种调试启动方式：

#### 1. Launch modelscope-balance
- **用途**：标准启动方式
- **特点**：直接运行 main.go
- **使用场景**：常规调试

#### 2. Launch with custom config
- **用途**：使用自定义配置文件启动
- **特点**：设置 CONFIG_FILE 环境变量
- **使用场景**：需要使用不同配置文件时

#### 3. Launch with debug config
- **用途**：调试模式启动
- **特点**：设置 DEBUG 环境变量
- **使用场景**：需要详细调试信息时

#### 4. Attach to process
- **用途**：附加到正在运行的进程
- **特点**：可以附加到已启动的服务
- **使用场景**：调试生产环境或已运行的服务

#### 5. Test current file
- **用途**：测试当前文件
- **特点**：运行测试文件
- **使用场景**：单元测试调试

### tasks.json 配置说明

项目已配置了多种任务：

#### 构建任务
- **Build modelscope-balance**：编译程序
- **Clean build**：清理构建文件
- **Full build cycle**：完整构建周期（整理依赖、格式化、检查、构建）

#### 运行任务
- **Run modelscope-balance**：运行程序
- **Build and Run**：构建并运行程序

#### 代码质量任务
- **Tidy dependencies**：整理依赖
- **Format code**：格式化代码
- **Vet code**：检查代码问题

#### 测试任务
- **Test modelscope-balance**：运行测试

## 调试步骤

### 1. 设置断点

在代码中点击行号左侧设置断点：

```go
// 在 main.go 中设置断点
func main() {
    config, err := LoadConfig("config.json") // 在这里设置断点
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }
    // ...
}
```

### 2. 启动调试

#### 方法一：使用调试面板
1. 按 `F5` 或点击调试面板
2. 选择启动配置
3. 开始调试

#### 方法二：使用命令面板
1. 按 `Ctrl+Shift+P` (Windows/Linux) 或 `Cmd+Shift+P` (macOS)
2. 输入 "Debug: Start Debugging"
3. 选择启动配置

#### 方法三：使用快捷键
- `F5`：启动调试
- `Ctrl+F5`：不调试启动
- `Shift+F5`：停止调试
- `F9`：切换断点
- `F10`：单步跳过
- `F11`：单步进入
- `Shift+F11`：单步退出

### 3. 调试控制

#### 断点控制
- **启用/禁用断点**：点击断点圆点
- **条件断点**：右键断点，设置条件
- **日志点**：右键断点，设置为日志点

#### 变量查看
- **变量面板**：查看当前作用域变量
- **监视面板**：添加表达式监视
- **悬停查看**：鼠标悬停查看变量值

#### 调用堆栈
- **调用堆栈面板**：查看函数调用链
- **切换堆栈**：点击不同堆栈帧

### 4. 常用调试场景

#### 场景1：调试配置加载
```go
// 在 LoadConfig 函数中设置断点
func LoadConfig(filename string) (*Config, error) {
    data, err := os.ReadFile(filename) // 断点
    if err != nil {
        return nil, err
    }
    // ...
}
```

#### 场景2：调试负载均衡逻辑
```go
// 在 GetNextAPIKey 函数中设置断点
func (lb *LoadBalancer) GetNextAPIKey() (string, error) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    for i := 0; i < len(lb.config.APIKeys); i++ {
        key := lb.config.APIKeys[lb.currentIndex] // 断点
        // ...
    }
}
```

#### 场景3：调试健康检查
```go
// 在 healthCheck 函数中设置断点
func (lb *LoadBalancer) healthCheck() {
    for _, key := range lb.config.APIKeys {
        go func(apiKey string) {
            resp, err := client.Do(req) // 断点
            // ...
        }(key)
    }
}
```

#### 场景4：调试请求转发
```go
// 在 HTTP 处理器中设置断点
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    apiKey, err := lb.GetNextAPIKey() // 断点
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    // ...
})
```

## 调试技巧

### 1. 条件断点
在特定条件下触发断点：
```go
// 只在特定 API Key 时触发断点
if key == "specific_api_key" {
    // 设置条件断点
}
```

### 2. 日志点
不暂停程序，只输出日志：
```go
// 在负载均衡器中添加日志点
log.Printf("当前使用 API Key: %s", key)
```

### 3. 表式监视
监视复杂表达式：
```go
// 监视面板中添加
len(lb.config.APIKeys)  // API Key 数量
lb.healthyKeys[key]     // 特定 Key 的健康状态
```

### 4. 多线程调试
调试并发场景：
```go
// 在健康检查的 goroutine 中设置断点
go func(apiKey string) {
    // 设置断点，调试并发问题
}(key)
```

## 常见问题

### 1. 调试器无法启动
**问题**：点击调试按钮无响应
**解决**：
- 检查 Go 扩展是否安装
- 检查 Go 工具链是否正确安装
- 重启 VSCode

### 2. 断点不生效
**问题**：设置了断点但不触发
**解决**：
- 确保代码已编译
- 检查断点位置是否在可执行代码行
- 尝试重新编译

### 3. 变量值不显示
**问题**：变量面板中看不到变量值
**解决**：
- 确保程序在断点处暂停
- 检查变量是否在当前作用域
- 尝试重新启动调试

### 4. 环境变量问题
**问题**：配置文件找不到
**解决**：
- 使用 "Launch with custom config" 配置
- 检查工作目录设置
- 确认配置文件路径正确

## 性能调试

### 1. 性能分析
使用 Go 的 pprof 工具：
```go
import _ "net/http/pprof"

// 在 main 函数中添加
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

### 2. 内存分析
查看内存使用情况：
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

### 3. CPU 分析
查看 CPU 使用情况：
```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

## 远程调试

如果需要在远程服务器上调试：

### 1. 编译带调试信息的程序
```bash
go build -gcflags="all=-N -l" -o modelscope-balance main.go
```

### 2. 在远程服务器上运行
```bash
./modelscope-balance
```

### 3. 配置远程调试
在 launch.json 中添加远程调试配置：
```json
{
    "name": "Remote Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "remotePath": "${workspaceFolder}",
    "port": 2345,
    "host": "remote-server-ip"
}
```

## 总结

通过 VSCode 的调试功能，可以方便地调试魔搭 API 负载均衡服务的各个方面：

1. **配置加载**：调试配置文件读取和解析
2. **负载均衡**：调试 API Key 选择逻辑
3. **健康检查**：调试健康检查机制
4. **请求转发**：调试 HTTP 请求处理
5. **并发问题**：调试多线程场景

合理使用断点、监视、日志点等调试工具，可以快速定位和解决问题。