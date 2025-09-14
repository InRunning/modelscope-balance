package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

// LoadBalancer 负载均衡器结构体
type LoadBalancer struct {
	apiKeys      []string
	currentIndex int
	mu           sync.Mutex
	healthyKeys  map[string]bool
	failedUntil  map[string]time.Time // 记录API key失效直到何时
	rateLimit    map[string]int       // 简单的请求计数
}

// NewLoadBalancer 创建新的负载均衡器
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		currentIndex: 0,
		healthyKeys:  make(map[string]bool),
		failedUntil:  make(map[string]time.Time),
		rateLimit:    make(map[string]int),
	}
}

// UpdateAPIKeys 更新API Keys列表
func (lb *LoadBalancer) UpdateAPIKeys(keys []string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.apiKeys = keys
	// 初始化所有API Key为健康状态
	for _, key := range keys {
		if _, exists := lb.healthyKeys[key]; !exists {
			lb.healthyKeys[key] = true
		}
	}
}

// GetNextAPIKey 获取下一个可用的API Key
func (lb *LoadBalancer) GetNextAPIKey() (string, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.apiKeys) == 0 {
		return "", fmt.Errorf("没有可用的API Key")
	}

	// 收集所有可用的API Keys（健康且不在失效期内）
	var availableKeys []string
	for _, key := range lb.apiKeys {
		if lb.healthyKeys[key] {
			// 检查是否在失效期内
			if failedUntil, exists := lb.failedUntil[key]; exists {
				if time.Now().Before(failedUntil) {
					continue // 跳过仍在失效期内的key
				}
				// 失效期已过，移除失效标记
				delete(lb.failedUntil, key)
			}
			availableKeys = append(availableKeys, key)
		}
	}

	if len(availableKeys) == 0 {
		return "", fmt.Errorf("没有可用的API Key")
	}

	// 随机选择一个可用的API Key
	selectedKey := availableKeys[rand.Intn(len(availableKeys))]
	lb.rateLimit[selectedKey]++
	return selectedKey, nil
}

// MarkAPIKeyFailed 标记API Key为失效，失效时间为10秒
func (lb *LoadBalancer) MarkAPIKeyFailed(key string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.failedUntil[key] = time.Now().Add(10 * time.Second)
	// 安全地显示API Key的前10个字符
	if len(key) > 10 {
		log.Printf("API Key %s... 已标记为失效，10秒后恢复", key[:10])
	} else {
		log.Printf("API Key %s 已标记为失效，10秒后恢复", key)
	}
}

// GetStats 获取统计信息
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	stats := make(map[string]interface{})
	for _, key := range lb.apiKeys {
		keyShort := key[:10] + "..."
		failedUntil, hasFailed := lb.failedUntil[key]
		stat := map[string]interface{}{
			"healthy":  lb.healthyKeys[key],
			"requests": lb.rateLimit[key],
		}
		if hasFailed {
			stat["failed_until"] = failedUntil
		}
		stats[keyShort] = stat
	}
	return stats
}

// createProxy 创建反向代理 - 支持真正的流式传输
func createProxy(target *url.URL, apiKey string, lb *LoadBalancer) http.Handler {
	// 创建自定义的Transport以处理HTTPS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // 保持SSL验证
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 保存原始请求体
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 记录请求信息 - 显示实际转发的目标URL
		log.Printf("转发请求到: %s%s", target.Host, r.URL.Path)
		// 安全地显示API Key的前10个字符
		if len(apiKey) > 10 {
			log.Printf("使用API Key: %s...", apiKey[:10])
		} else {
			log.Printf("使用API Key: %s", apiKey)
		}
		log.Printf("完整目标URL: %s", r.URL.String())

		// 记录请求方法
		log.Printf("请求方法: %s", r.Method)

		// 记录请求头
		log.Printf("请求头: %v", r.Header)

		// 记录请求体（仅记录前100个字符）
		if bodyBytes != nil {
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 100 {
				bodyStr = bodyStr[:100] + "..."
			}
			log.Printf("请求体: %s", bodyStr)
		}

		// 创建新的请求URL
		targetURL := &url.URL{
			Scheme:   target.Scheme,
			Host:     target.Host,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}

		// 创建新的请求
		req, err := http.NewRequest(r.Method, targetURL.String(), bytes.NewBuffer(bodyBytes))
		if err != nil {
			log.Printf("创建请求失败: %v", err)
			http.Error(w, "创建请求失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 复制请求头
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// 设置Host头
		req.Host = target.Host
		req.Header.Set("Host", target.Host)

		// 设置Authorization头
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// 发送请求
		resp, err := transport.RoundTrip(req)
		if err != nil {
			log.Printf("请求失败: %v", err)
			http.Error(w, "请求失败: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		log.Printf("收到响应状态码: %d", resp.StatusCode)
		
		// 处理API Key失效情况
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			// API Key无效或权限不足，标记为失效
			if len(apiKey) > 10 {
				log.Printf("API Key %s... 无效，标记为失效", apiKey[:10])
			} else {
				log.Printf("API Key %s 无效，标记为失效", apiKey)
			}
			lb.MarkAPIKeyFailed(apiKey)
		} else if resp.StatusCode >= 400 {
			log.Printf("API返回错误状态码: %d", resp.StatusCode)
		}

		// 复制响应头
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// 设置状态码
		w.WriteHeader(resp.StatusCode)

		// 对于错误响应，读取完整响应体用于日志记录
		if resp.StatusCode >= 400 {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil {
				log.Printf("错误响应体: %s", string(bodyBytes))
				w.Write(bodyBytes)
			} else {
				log.Printf("读取错误响应体失败: %v", err)
			}
			return
		}

		// 检查是否为流式JSON响应（如SSE或流式API）
		contentType := resp.Header.Get("Content-Type")
		isStream := strings.Contains(contentType, "text/event-stream") ||
				   strings.Contains(contentType, "application/x-ndjson") ||
				   strings.Contains(contentType, "application/jsonl")
		
		// 对于成功的响应，启用流式传输
		// 确保响应是流式的，不缓冲
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		if isStream {
			// 对于流式响应（SSE、NDJSON等），逐行处理
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					// 写入行数据并立即刷新
					w.Write([]byte(line))
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
			}
			
			if err := scanner.Err(); err != nil {
				log.Printf("扫描流式响应失败: %v", err)
			}
		} else {
			// 对于普通响应（包括JSON），使用标准的流式传输
			// 不进行复杂的JSON解析，直接转发原始数据
			buf := make([]byte, 4*1024) // 4KB缓冲区，更小的缓冲区以实现更实时的响应
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					// 写入数据并立即刷新
					w.Write(buf[:n])
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Printf("流式传输响应体失败: %v", err)
					break
				}
			}
		}
	})
}

// parseAPIKeysFromHeader 从Authorization头解析API Keys
func parseAPIKeysFromHeader(authHeader string) ([]string, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("Authorization头为空")
	}

	// 移除"Bearer "前缀（如果有）
	if strings.HasPrefix(authHeader, "Bearer ") {
		authHeader = authHeader[7:]
	}

	// 按逗号分割API Keys
	keys := strings.Split(authHeader, ",")

	// 清理每个key的空格
	var cleanKeys []string
	for _, key := range keys {
		cleanKey := strings.TrimSpace(key)
		if cleanKey != "" {
			cleanKeys = append(cleanKeys, cleanKey)
		}
	}

	if len(cleanKeys) == 0 {
		return nil, fmt.Errorf("没有找到有效的API Keys")
	}

	return cleanKeys, nil
}

func main() {
	// 读取配置文件
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var config struct {
		ServerPort string `json:"server_port"`
		TargetURL  string `json:"target_url"`
		HealthCheck bool `json:"health_check"`
		HealthPath string `json:"health_path"`
	}
	
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 创建负载均衡器
	lb := NewLoadBalancer()

	// 从配置文件读取目标URL
	targetURL, err := url.Parse(config.TargetURL)
	if err != nil {
		log.Fatalf("解析目标URL失败: %v", err)
	}

	// 创建HTTP处理器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 从Authorization头获取API Keys
		authHeader := r.Header.Get("Authorization")
		apiKeys, err := parseAPIKeysFromHeader(authHeader)
		if err != nil {
			http.Error(w, fmt.Sprintf("解析API Keys失败: %v", err), http.StatusBadRequest)
			return
		}

		// 更新负载均衡器的API Keys
		lb.UpdateAPIKeys(apiKeys)

		// 获取下一个API Key
		apiKey, err := lb.GetNextAPIKey()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// 创建代理
		proxy := createProxy(targetURL, apiKey, lb)

		// 处理请求
		proxy.ServeHTTP(w, r)
	})

	// 添加统计信息端点
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := lb.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// 添加健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		hasAvailableKeys := false
		lb.mu.Lock()
		for _, key := range lb.apiKeys {
			if lb.healthyKeys[key] {
				// 检查是否在失效期内
				if failedUntil, exists := lb.failedUntil[key]; !exists || time.Now().After(failedUntil) {
					hasAvailableKeys = true
					break
				}
			}
		}
		lb.mu.Unlock()

		if hasAvailableKeys {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Service is healthy")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Service is unhealthy")
		}
	})

	// 启动服务器
	addr := ":" + config.ServerPort
	log.Printf("启动无状态代理服务器，监听地址: %s", addr)
	log.Printf("目标URL: %s", targetURL.String())
	log.Printf("健康检查: %v", config.HealthCheck)
	log.Printf("健康检查路径: %s", config.HealthPath)
	log.Printf("请使用Authorization头传递API Keys，格式：Bearer key1,key2,key3...")

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}