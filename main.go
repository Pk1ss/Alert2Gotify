package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Alertmanager 推送的 JSON 结构定义
type AlertmanagerPayload struct {
	Status  string  `json:"status"` // firing 或 resolved
	Alerts  []Alert `json:"alerts"` // 告警列表
	Version string  `json:"version"`
}

// 单条告警的结构
type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`      // 标签（alertname、severity 等）
	Annotations map[string]string `json:"annotations"` // 注释（summary、description 等）
	StartsAt    string            `json:"startsAt"`    // 触发时间
	EndsAt      string            `json:"endsAt"`      // 恢复时间（未恢复时为 0001-01-01...）
}

// Gotify 接收的 JSON 结构
type GotifyPayload struct {
	Title    string `json:"title"`    // 消息标题
	Message  string `json:"message"`  // 消息内容
	Priority int    `json:"priority"` // 优先级（1-10）
}

func main() {
	// 从环境变量获取配置（方便部署时修改，无需改代码）
	gotifyURL := os.Getenv("GOTIFY_URL")
	gotifyToken := os.Getenv("GOTIFY_TOKEN")
	listenPort := os.Getenv("LISTEN_PORT")

	// 默认值（如果环境变量未设置）
	if gotifyURL == "" {
		gotifyURL = "http://localhost:8080" // 替换为你的 Gotify 地址
	}
	if gotifyToken == "" {
		gotifyToken = "your-token-here" // 替换为你的 Gotify 令牌
	}
	if listenPort == "" {
		listenPort = "8081" // 中间件监听端口
	}

	// 注册 HTTP 处理函数，接收 Alertmanager 的请求
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// 1. 解析 Alertmanager 发送的 JSON 数据
		var alertData AlertmanagerPayload
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&alertData); err != nil {
			http.Error(w, "解析请求失败: "+err.Error(), http.StatusBadRequest)
			fmt.Printf("解析失败: %v\n", err)
			return
		}

		// 2. 遍历每条告警，转换格式并推送到 Gotify
		for _, alert := range alertData.Alerts {
			// 提取关键信息（处理空值，避免 panic）
			alertName := getMapValue(alert.Labels, "alertname", "未知告警")
			severity := getMapValue(alert.Labels, "severity", "unknown")
			summary := getMapValue(alert.Annotations, "summary", "无摘要")
			description := getMapValue(alert.Annotations, "description", "无详情")
			status := strings.ToUpper(alert.Status) // 转为大写（FIRING/RESOLVED）

			// 3. 映射优先级（根据 severity 调整）
			priority := 5 // 默认优先级
			switch strings.ToLower(severity) {
			case "critical":
				priority = 8 // 严重告警优先级高
			case "warning":
				priority = 5
			case "info":
				priority = 3 // 信息类优先级低
			}

			// 4. 格式化时间（将 ISO 8601 转为本地时间，更易读）
			startTime := formatTime(alert.StartsAt)
			endTime := formatTime(alert.EndsAt)
			timeStr := fmt.Sprintf("触发时间: %s", startTime)
			if status == "RESOLVED" && endTime != "" {
				timeStr = fmt.Sprintf("恢复时间: %s", endTime)
			}

			// 5. 构造 Gotify 消息
			gotifyMsg := GotifyPayload{
				Title:    fmt.Sprintf("[%s] %s (%s)", severity, alertName, status),
				Message:  fmt.Sprintf("%s\n%s\n%s", summary, description, timeStr),
				Priority: priority,
			}

			// 6. 发送到 Gotify
			resp, err := sendToGotify(gotifyURL, gotifyToken, gotifyMsg)
			if err != nil {
				fmt.Printf("推送失败: %v\n", err)
				continue
			}
			fmt.Printf("推送成功: %s，状态码: %d\n", alertName, resp.StatusCode)
		}

		// 回复 Alertmanager 表示处理成功
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// 启动服务
	fmt.Printf("中间件启动成功，监听端口: %s\n", listenPort)
	fmt.Printf("Gotify 地址: %s\n", gotifyURL)
	err := http.ListenAndServe(":"+listenPort, nil)
	if err != nil {
		fmt.Printf("服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

// 工具函数：安全获取 map 中的值（避免 key 不存在导致的 nil 错误）
func getMapValue(m map[string]string, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		return val
	}
	return defaultValue
}

// 工具函数：格式化时间（将 ISO 8601 转为本地时间）
func formatTime(isoTime string) string {
	if isoTime == "0001-01-01T00:00:00Z" { // 未恢复时的默认时间
		return ""
	}
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime // 解析失败则返回原始字符串
	}
	return t.Local().Format("2006-01-02 15:04:05") // 本地时间格式
}

// 工具函数：发送消息到 Gotify
func sendToGotify(gotifyURL, token string, msg GotifyPayload) (*http.Response, error) {
	// 序列化消息为 JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %v", err)
	}

	// 构造 Gotify 完整 URL（包含令牌）
	fullURL := fmt.Sprintf("%s/message?token=%s", gotifyURL, token)

	// 发送 POST 请求
	resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("请求发送失败: %v", err)
	}
	return resp, nil
}
