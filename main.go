package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

// Config 结构体用于存储配置信息
type Config struct {
	APIKeys     []string `json:"api_keys"`
	ServerPort  string   `json:"server_port"`
	TargetURL   string   `json:"target_url"`
	HealthCheck bool     `json:"health_check"`
	HealthPath  string   `json:"health_path"`
}

// LoadBalancer 负载均衡器结构体
type LoadBalancer struct {
	config       *Config
	currentIndex int
	mu           sync.Mutex
	healthyKeys  map[string]bool
	lastUsed     map[string]time.Time
	rateLimit    map[string]int // 简单的请求计数
}

// NewLoadBalancer 创建新的负载均衡器
func NewLoadBalancer(config *Config) *LoadBalancer {
	lb := &LoadBalancer{
		config:       config,
		currentIndex: 0,
		healthyKeys:  make(map[string]bool),
		lastUsed:     make(map[string]time.Time),
		rateLimit:    make(map[string]int),
	}

	// 初始化所有API Key为健康状态
	for _, key := range config.APIKeys {
		lb.healthyKeys[key] = true
	}

	return lb
}

// GetNextAPIKey 获取下一个可用的API Key
func (lb *LoadBalancer) GetNextAPIKey() (string, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 简单的负载均衡策略：轮询
	for i := 0; i < len(lb.config.APIKeys); i++ {
		key := lb.config.APIKeys[lb.currentIndex]
		lb.currentIndex = (lb.currentIndex + 1) % len(lb.config.APIKeys)

		// 检查API Key是否健康
		if lb.healthyKeys[key] {
			// 更新最后使用时间
			lb.lastUsed[key] = time.Now()
			lb.rateLimit[key]++
			return key, nil
		}
	}

	return "", fmt.Errorf("没有可用的API Key")
}

// MarkAPIKeyUnhealthy 标记API Key为不健康
func (lb *LoadBalancer) MarkAPIKeyUnhealthy(key string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.healthyKeys[key] = false
	log.Printf("API Key %s 已标记为不健康", key[:10]+"...")
}

// MarkAPIKeyHealthy 标记API Key为健康
func (lb *LoadBalancer) MarkAPIKeyHealthy(key string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.healthyKeys[key] = true
	log.Printf("API Key %s 已标记为健康", key[:10]+"...")
}

// GetStats 获取统计信息
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	stats := make(map[string]interface{})
	for _, key := range lb.config.APIKeys {
		keyShort := key[:10] + "..."
		stats[keyShort] = map[string]interface{}{
			"healthy":   lb.healthyKeys[key],
			"last_used": lb.lastUsed[key],
			"requests":  lb.rateLimit[key],
		}
	}
	return stats
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	if config.ServerPort == "" {
		config.ServerPort = "8080"
	}
	if config.TargetURL == "" {
		config.TargetURL = "https://api-inference.modelscope.cn"
	}
	if config.HealthPath == "" {
		config.HealthPath = "/v1/models"
	}

	return &config, nil
}

// createProxy 创建反向代理
func createProxy(target *url.URL, apiKey string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 配置Transport以处理HTTPS
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // 保持SSL验证
		},
	}

	// 修改请求以添加认证头
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 保存原始请求体
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		originalDirector(req)

		// 确保Host头正确设置为目标服务器
		req.Host = target.Host
		req.Header.Set("Host", target.Host)

		// 魔搭API使用单个API Key
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// 恢复请求体
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 记录请求信息 - 显示实际转发的目标URL
		log.Printf("转发请求到: %s%s", target.Host, req.URL.Path)
		log.Printf("使用API Key: %s...", apiKey[:10])
		log.Printf("完整目标URL: %s", req.URL.String())

		// 记录请求方法
		log.Printf("请求方法: %s", req.Method)

		// 记录请求头
		log.Printf("请求头: %v", req.Header)

		// 记录请求体（仅记录前100个字符）
		if bodyBytes != nil {
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 100 {
				bodyStr = bodyStr[:100] + "..."
			}
			log.Printf("请求体: %s", bodyStr)
		}
	}

	// 修改响应处理
	proxy.ModifyResponse = func(resp *http.Response) error {
		log.Printf("收到响应状态码: %d", resp.StatusCode)
		if resp.StatusCode >= 400 {
			log.Printf("API返回错误状态码: %d", resp.StatusCode)
			// 读取响应体以获取更多错误信息
			if resp.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err == nil {
					log.Printf("错误响应体: %s", string(bodyBytes))
					// 重新设置响应体
					resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}
		}
		return nil
	}

	// 添加错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("代理错误: %v", err)
		http.Error(w, "代理转发失败: "+err.Error(), http.StatusBadGateway)
	}

	return proxy
}

// healthCheck 健康检查函数
func (lb *LoadBalancer) healthCheck() {
	if !lb.config.HealthCheck {
		return
	}

	for _, key := range lb.config.APIKeys {
		go func(apiKey string) {
			client := &http.Client{Timeout: 10 * time.Second}
			targetURL, _ := url.Parse(lb.config.TargetURL)

			// 正确构建健康检查URL
			healthURL := targetURL.ResolveReference(&url.URL{Path: lb.config.HealthPath})
			req, err := http.NewRequest("GET", healthURL.String(), nil)
			if err != nil {
				log.Printf("创建健康检查请求失败: %v", err)
				return
			}

			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := client.Do(req)
			if err != nil || resp.StatusCode >= 500 {
				lb.MarkAPIKeyUnhealthy(apiKey)
			} else {
				lb.MarkAPIKeyHealthy(apiKey)
			}

			if resp != nil {
				resp.Body.Close()
			}
		}(key)
	}
}

func main() {
	// 加载配置
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if len(config.APIKeys) == 0 {
		log.Fatal("配置中没有找到API Keys")
	}

	log.Printf("加载了 %d 个API Keys", len(config.APIKeys))

	// 创建负载均衡器
	lb := NewLoadBalancer(config)

	// 启动健康检查
	if config.HealthCheck {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				lb.healthCheck()
			}
		}()
	}

	// 解析目标URL
	targetURL, err := url.Parse(config.TargetURL)
	if err != nil {
		log.Fatalf("解析目标URL失败: %v", err)
	}

	// 创建HTTP处理器
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 获取下一个API Key
		apiKey, err := lb.GetNextAPIKey()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// 创建代理
		proxy := createProxy(targetURL, apiKey)

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
		healthy := false
		for _, isHealthy := range lb.healthyKeys {
			if isHealthy {
				healthy = true
				break
			}
		}

		if healthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Service is healthy")
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Service is unhealthy")
		}
	})

	// 启动服务器
	addr := ":" + config.ServerPort
	log.Printf("启动负载均衡服务器，监听地址: %s", addr)
	log.Printf("目标URL: %s", config.TargetURL)

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
