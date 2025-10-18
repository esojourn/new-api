package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// NotificationService Webhook通知服务
type NotificationService struct {
	httpClient *http.Client
}

// NewNotificationService 创建通知服务实例
func NewNotificationService() *NotificationService {
	return &NotificationService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendWebhookNotification 发送Webhook通知
func SendWebhookNotification(payload interface{}, webhookUrl string, secret string) error {
	// Marshal payload to JSON
	payloadData, err := json.Marshal(payload)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to marshal webhook payload: %v", err))
		return err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(payloadData))
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to create webhook request: %v", err))
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "New-API-Monitor/1.0")

	// Add signature if secret is provided
	if secret != "" {
		signature := generateHmacSignature(payloadData, secret)
		req.Header.Set("X-Signature", signature)
		req.Header.Set("X-Signature-Algorithm", "sha256")
	}

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		common.SysLog(fmt.Sprintf("webhook request failed for %s: %v", webhookUrl, err))
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		common.SysLog(fmt.Sprintf("webhook request failed for %s: status=%d, body=%s", webhookUrl, resp.StatusCode, string(body)))
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	common.SysLog(fmt.Sprintf("webhook notification sent successfully: %s", webhookUrl))
	return nil
}

// TestWebhookNotification 测试Webhook通知
func TestWebhookNotification(webhookUrl string, secret string) (map[string]interface{}, error) {
	testPayload := map[string]interface{}{
		"event":      "monitor_test",
		"timestamp":  time.Now().Format(time.RFC3339),
		"message":    "This is a test webhook notification",
		"task_id":    0,
		"task_name":  "Test Webhook",
		"channel_id": 0,
		"model":      "test-model",
		"status":     "success",
	}

	// Send request
	startTime := time.Now()
	err := SendWebhookNotification(testPayload, webhookUrl, secret)
	duration := time.Since(startTime)

	if err != nil {
		return map[string]interface{}{
			"success":       false,
			"error":         err.Error(),
			"response_time": int(duration.Milliseconds()),
		}, err
	}

	return map[string]interface{}{
		"success":       true,
		"error":         nil,
		"response_time": int(duration.Milliseconds()),
		"message":       "Webhook test successful",
	}, nil
}

// SendWebhookNotificationAsync 异步发送Webhook通知
func SendWebhookNotificationAsync(payload interface{}, webhookUrl string, secret string) {
	go func() {
		if err := SendWebhookNotification(payload, webhookUrl, secret); err != nil {
			common.SysLog(fmt.Sprintf("async webhook notification failed: %v", err))
		}
	}()
}

// generateHmacSignature 生成HMAC签名
func generateHmacSignature(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyWebhookSignature 验证Webhook签名
func VerifyWebhookSignature(data []byte, signature string, secret string) bool {
	expected := generateHmacSignature(data, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// DingTalkNotification 钉钉通知
func SendDingTalkNotification(webhookUrl string, title string, content string, status string) error {
	// 0=success, 1=failed
	color := "#0099FF"
	if status == "failed" {
		color = "#FF0000"
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  content,
		},
	}

	return SendWebhookNotification(payload, webhookUrl, "")
}

// SlackNotification Slack通知
func SendSlackNotification(webhookUrl string, taskName string, channelName string, modelName string, status string, errorMsg string) error {
	color := "good"
	statusText := "✅ Success"
	if status == "failed" {
		color = "danger"
		statusText = "❌ Failed"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": fmt.Sprintf("Monitor Alert: %s", taskName),
				"fields": []map[string]string{
					{
						"title": "Status",
						"value": statusText,
						"short": true,
					},
					{
						"title": "Channel",
						"value": channelName,
						"short": true,
					},
					{
						"title": "Model",
						"value": modelName,
						"short": true,
					},
					{
						"title": "Timestamp",
						"value": time.Now().Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	if errorMsg != "" {
		attachments := payload["attachments"].([]map[string]interface{})
		fields := attachments[0]["fields"].([]map[string]string)
		fields = append(fields, map[string]string{
			"title": "Error",
			"value": errorMsg,
			"short": false,
		})
		attachments[0]["fields"] = fields
	}

	return SendWebhookNotification(payload, webhookUrl, "")
}

// CustomNotification 自定义通知格式
func SendCustomNotification(webhookUrl string, customPayload map[string]interface{}, secret string) error {
	return SendWebhookNotification(customPayload, webhookUrl, secret)
}

// NotificationTemplate 通知模板
type NotificationTemplate struct {
	Type     string                 `json:"type"` // webhook, dingtalk, slack, custom
	Format   map[string]interface{} `json:"format"`
	Webhook  string                 `json:"webhook"`
	Secret   string                 `json:"secret,omitempty"`
}

// SendNotificationByTemplate 按模板发送通知
func SendNotificationByTemplate(template *NotificationTemplate, data map[string]interface{}) error {
	switch template.Type {
	case "webhook":
		return SendWebhookNotification(data, template.Webhook, template.Secret)
	case "dingtalk":
		title, _ := data["title"].(string)
		content, _ := data["content"].(string)
		status, _ := data["status"].(string)
		return SendDingTalkNotification(template.Webhook, title, content, status)
	case "slack":
		taskName, _ := data["task_name"].(string)
		channelName, _ := data["channel_name"].(string)
		modelName, _ := data["model"].(string)
		status, _ := data["status"].(string)
		errorMsg, _ := data["error_message"].(string)
		return SendSlackNotification(template.Webhook, taskName, channelName, modelName, status, errorMsg)
	case "custom":
		return SendCustomNotification(template.Webhook, data, template.Secret)
	default:
		return fmt.Errorf("unknown notification type: %s", template.Type)
	}
}
