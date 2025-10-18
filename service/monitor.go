package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/types"
	"github.com/robfig/cron/v3"
)

var (
	monitorTaskSchedulers = make(map[int]*cron.Cron)
	monitorLock           sync.Mutex
	globalCron            *cron.Cron
)

// MonitorService 监控服务
type MonitorService struct {
	notificationService *NotificationService
}

// NewMonitorService 创建监控服务实例
func NewMonitorService(notifService *NotificationService) *MonitorService {
	return &MonitorService{
		notificationService: notifService,
	}
}

// InitMonitorSystem 初始化监控系统
func InitMonitorSystem() error {
	// Initialize database tables
	if err := model.InitMonitorTables(); err != nil {
		common.SysLog(fmt.Sprintf("failed to initialize monitor tables: %v", err))
		return err
	}

	// Create global cron scheduler
	globalCron = cron.New()

	// Load and schedule enabled tasks
	if err := ReloadAllMonitorTasks(); err != nil {
		common.SysLog(fmt.Sprintf("failed to reload monitor tasks: %v", err))
		return err
	}

	// Add cleanup job - run daily at 2 AM to clean data older than 7 days
	_, err := globalCron.AddFunc("0 2 * * *", func() {
		if err := model.CleanOldMonitorResults(7); err != nil {
			common.SysLog(fmt.Sprintf("failed to clean old monitor results: %v", err))
		}
	})
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to add cleanup job: %v", err))
	}

	// Start the global cron
	globalCron.Start()

	common.SysLog("Monitor system initialized successfully")
	return nil
}

// StopMonitorSystem 优雅关闭监控系统
func StopMonitorSystem() {
	if globalCron != nil {
		globalCron.Stop()
	}

	monitorLock.Lock()
	defer monitorLock.Unlock()

	for _, c := range monitorTaskSchedulers {
		c.Stop()
	}
	monitorTaskSchedulers = make(map[int]*cron.Cron)
}

// ReloadAllMonitorTasks 重新加载所有监控任务
func ReloadAllMonitorTasks() error {
	tasks, err := model.GetEnabledMonitorTasks()
	if err != nil {
		return err
	}

	// Remove old schedulers
	monitorLock.Lock()
	for _, c := range monitorTaskSchedulers {
		c.Stop()
	}
	monitorTaskSchedulers = make(map[int]*cron.Cron)
	monitorLock.Unlock()

	// Schedule new tasks
	for _, task := range tasks {
		if err := scheduleMonitorTask(task); err != nil {
			common.SysLog(fmt.Sprintf("failed to schedule monitor task %d: %v", task.Id, err))
		}
	}

	return nil
}

// scheduleMonitorTask 为单个任务配置定时任务
func scheduleMonitorTask(task *model.MonitorTask) error {
	schedule, err := task.GetSchedule()
	if err != nil {
		return err
	}

	// Generate cron expression
	var cronExpr string
	if schedule.CronExpr != "" {
		cronExpr = schedule.CronExpr
	} else if schedule.Hour != nil && schedule.Minute != nil {
		// Daily at specific time
		cronExpr = fmt.Sprintf("%d %d * * *", *schedule.Minute, *schedule.Hour)
	} else {
		// Fixed interval
		if schedule.Interval <= 0 {
			schedule.Interval = 300 // default 5 minutes
		}
		cronExpr = fmt.Sprintf("@every %ds", schedule.Interval)
	}

	// Create a new cron scheduler for this task
	c := cron.New()

	// Add job
	_, err = c.AddFunc(cronExpr, func() {
		if err := RunMonitorTask(task); err != nil {
			common.SysLog(fmt.Sprintf("failed to run monitor task %d: %v", task.Id, err))
		}
	})
	if err != nil {
		return err
	}

	monitorLock.Lock()
	defer monitorLock.Unlock()

	// Stop old scheduler if exists
	if oldCron, exists := monitorTaskSchedulers[task.Id]; exists {
		oldCron.Stop()
	}

	monitorTaskSchedulers[task.Id] = c
	c.Start()

	common.SysLog(fmt.Sprintf("scheduled monitor task %d with cron expression: %s", task.Id, cronExpr))
	return nil
}

// RunMonitorTask 执行监控任务
func RunMonitorTask(task *model.MonitorTask) error {
	// Update last run time and status
	task.LastRunAt = common.GetTimestamp()
	task.LastRunStatus = 2 // running

	if err := task.Update(); err != nil {
		return err
	}

	// Get channels and models
	channelIds, _ := task.GetChannelIds()
	models, _ := task.GetModels()

	if len(channelIds) == 0 || len(models) == 0 {
		common.SysLog(fmt.Sprintf("monitor task %d has no channels or models configured", task.Id))
		task.LastRunStatus = 1 // failed
		task.FailureCount++
		_ = task.Update()
		return fmt.Errorf("no channels or models configured")
	}

	// Test each channel-model combination
	successCount := 0
	failureCount := 0

	for _, channelId := range channelIds {
		for _, modelName := range models {
			result, err := testChannelModel(task, channelId, modelName)
			if err != nil {
				common.SysLog(fmt.Sprintf("failed to test channel %d model %s: %v", channelId, modelName, err))
				failureCount++
			} else {
				// Save result
				if err := result.Save(); err != nil {
					common.SysLog(fmt.Sprintf("failed to save monitor result: %v", err))
				}

				if result.Status == 0 {
					successCount++
				} else {
					failureCount++
				}

				// Check alerts and send notifications
				if result.Status != 0 { // failure
					_ = checkAndSendAlerts(task, result)
				}
			}
		}
	}

	// Update task statistics
	task.TotalRuns++
	task.TotalSuccesses += int64(successCount)
	task.TotalFailures += int64(failureCount)

	if failureCount > 0 {
		task.FailureCount++
		task.SuccessCount = 0
		task.LastRunStatus = 1 // failed
	} else {
		task.FailureCount = 0
		task.SuccessCount++
		task.LastRunStatus = 0 // success
	}

	// Calculate average response time
	results, _ := model.GetMonitorResultsByTaskId(task.Id, 100)
	if len(results) > 0 {
		var totalTime int64
		for _, r := range results {
			totalTime += int64(r.ResponseTime)
		}
		task.AvgResponseTime = int(totalTime / int64(len(results)))
	}

	if err := task.Update(); err != nil {
		return err
	}

	common.SysLog(fmt.Sprintf("monitor task %d completed: successes=%d, failures=%d", task.Id, successCount, failureCount))
	return nil
}

// testChannelModel 测试单个渠道的单个模型
func testChannelModel(task *model.MonitorTask, channelId int, modelName string) (*model.MonitorResult, error) {
	result := &model.MonitorResult{
		TaskId:    task.Id,
		ChannelId: channelId,
		Model:     modelName,
		CreatedAt: common.GetTimestamp(),
	}

	// Get channel
	channel, err := model.GetChannelById(channelId, false)
	if err != nil {
		result.Status = 1
		result.ErrorMessage = "channel not found or disabled"
		return result, nil
	}

	// Check if channel is enabled
	if channel.Status != 1 {
		result.Status = 1
		result.ErrorMessage = "channel is disabled"
		return result, nil
	}

	// Create test request
	startTime := time.Now()
	response, err := sendTestChatRequest(channel, modelName, task.TestContent, task.Timeout)
	duration := time.Since(startTime)

	result.ResponseTime = int(duration.Milliseconds())

	if err != nil {
		result.Status = 1
		result.ErrorMessage = err.Error()
		result.RetryCount = 0

		// Retry logic
		for i := 0; i < task.MaxRetries; i++ {
			result.RetryCount++
			time.Sleep(time.Second * 2) // wait 2 seconds before retry

			startTime := time.Now()
			response, err = sendTestChatRequest(channel, modelName, task.TestContent, task.Timeout)
			duration := time.Since(startTime)

			result.ResponseTime = int(duration.Milliseconds())

			if err == nil {
				break
			}
			result.ErrorMessage = err.Error()
		}

		if err != nil {
			return result, nil
		}
	}

	// Validate response
	if task.ExpectedPattern != "" && response != "" {
		matched, err := regexp.MatchString(task.ExpectedPattern, response)
		if err != nil || !matched {
			result.Status = 1
			if err != nil {
				result.ErrorMessage = fmt.Sprintf("pattern matching error: %v", err)
			} else {
				result.ErrorMessage = "response does not match expected pattern"
			}
			return result, nil
		}
	}

	result.Status = 0 // success
	result.FullResponse = response
	return result, nil
}

// sendTestChatRequest 发送实际的对话测试请求 - 使用同步HTTP请求方式
func sendTestChatRequest(channel *model.Channel, modelName string, content string, timeout int) (string, error) {
	// 获取用户信息用于获取Quota
	user, err := model.GetUserByID(1) // Monitor user ID
	if err != nil {
		return "", fmt.Errorf("failed to get monitor user: %w", err)
	}

	// Create OpenAI-format request
	requestBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": content,
			},
		},
		"temperature": 0.7,
		"max_tokens":  100,
	}

	requestData, _ := json.Marshal(requestBody)

	// 为了确保使用正确的API key认证，直接使用频道的API key
	// 创建HTTP请求到本地API
	apiURL := fmt.Sprintf("http://localhost:%d/v1/chat/completions", common.Port)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", user.Token))
	req.Header.Set("X-Monitor-Task", "true")
	req.Header.Set("X-Channel-ID", strconv.Itoa(channel.Id))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	var responseBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors in response
	if resp.StatusCode != http.StatusOK {
		if errMsg, ok := responseBody["error"]; ok {
			return "", fmt.Errorf("api error: %v", errMsg)
		}
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	// Extract content from response
	if choices, ok := responseBody["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid response format")
}

// checkAndSendAlerts 检查是否触发告警并发送通知
func checkAndSendAlerts(task *model.MonitorTask, result *model.MonitorResult) error {
	// Get alerts for this task
	alerts, err := model.GetMonitorAlertsByTaskId(task.Id)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		triggered := false

		if alert.AlertType == 0 { // consecutive failures
			if task.FailureCount >= alert.Threshold {
				triggered = true
			}
		} else if alert.AlertType == 1 { // low success rate
			// Get recent results within window
			now := common.GetTimestamp()
			windowStart := now - int64(alert.WindowSize)*60

			recentResults, _, _ := model.GetMonitorResultsByTaskIdAndDateRange(task.Id, windowStart, now, 1000, 0)

			if len(recentResults) > 0 {
				var successes int64
				for _, r := range recentResults {
					if r.Status == 0 {
						successes++
					}
				}
				successRate := float64(successes) / float64(len(recentResults)) * 100
				if successRate < float64(alert.Threshold) {
					triggered = true
				}
			}
		}

		if triggered {
			// Send notifications
			if alert.WebhookEnabled {
				webhooks, _ := model.GetMonitorWebhooksByTaskId(task.Id)
				for _, webhook := range webhooks {
					_ = SendWebhookNotification(&WebhookPayload{
						Event:       "monitor_failure",
						TaskId:      task.Id,
						TaskName:    task.Name,
						ChannelId:   result.ChannelId,
						Model:       result.Model,
						Status:      "failed",
						ErrorMsg:    result.ErrorMessage,
						ResponseTime: result.ResponseTime,
						Timestamp:   time.Now().Format(time.RFC3339),
						AlertType:   alert.AlertType,
					}, webhook.Url, webhook.Secret)

					// Mark as notified
					result.WebhookNotified = true
				}
			}

			// Record last trigger time
			alert.LastTriggeredTime = common.GetTimestamp()
			_ = alert.Save()
		}
	}

	return nil
}

// RunMonitorTaskNow 手动运行监控任务
func RunMonitorTaskNow(taskId int) ([]*model.MonitorResult, error) {
	task, err := model.GetMonitorTaskById(taskId)
	if err != nil {
		return nil, err
	}

	if err := RunMonitorTask(task); err != nil {
		return nil, err
	}

	// Return latest results
	return model.GetLatestMonitorResultsByTask(taskId)
}

// GetMonitorTaskStatus 获取监控任务状态
func GetMonitorTaskStatus(taskId int) (interface{}, error) {
	task, err := model.GetMonitorTaskById(taskId)
	if err != nil {
		return nil, err
	}

	stats, _ := model.GetMonitorTaskStatistics(taskId)

	return map[string]interface{}{
		"task":        task,
		"statistics":  stats,
		"last_run_at": task.LastRunAt,
		"status":      task.LastRunStatus,
	}, nil
}

// GetMonitorResults 获取监控结果
func GetMonitorResults(taskId int, limit int) ([]*model.MonitorResult, error) {
	return model.GetMonitorResultsByTaskId(taskId, limit)
}

// GetMonitorResultsInRange 获取时间范围内的监控结果
func GetMonitorResultsInRange(taskId int, startTime int64, endTime int64, limit int, offset int) ([]*model.MonitorResult, int64, error) {
	return model.GetMonitorResultsByTaskIdAndDateRange(taskId, startTime, endTime, limit, offset)
}

// CreateMonitorTask 创建监控任务
func CreateMonitorTask(task *model.MonitorTask) error {
	if err := task.Save(); err != nil {
		return err
	}

	// Schedule the task
	return scheduleMonitorTask(task)
}

// UpdateMonitorTask 更新监控任务
func UpdateMonitorTask(task *model.MonitorTask) error {
	if err := task.Update(); err != nil {
		return err
	}

	// Re-schedule if enabled
	if task.Enabled {
		return scheduleMonitorTask(task)
	} else {
		// Stop the scheduler
		monitorLock.Lock()
		defer monitorLock.Unlock()
		if c, exists := monitorTaskSchedulers[task.Id]; exists {
			c.Stop()
			delete(monitorTaskSchedulers, task.Id)
		}
	}

	return nil
}

// DeleteMonitorTask 删除监控任务
func DeleteMonitorTask(taskId int) error {
	task, err := model.GetMonitorTaskById(taskId)
	if err != nil {
		return err
	}

	// Stop the scheduler
	monitorLock.Lock()
	defer monitorLock.Unlock()
	if c, exists := monitorTaskSchedulers[taskId]; exists {
		c.Stop()
		delete(monitorTaskSchedulers, taskId)
	}

	return task.Delete()
}

// ToggleMonitorTask 启用/禁用监控任务
func ToggleMonitorTask(taskId int, enabled bool) error {
	task, err := model.GetMonitorTaskById(taskId)
	if err != nil {
		return err
	}

	task.Enabled = enabled

	if enabled {
		return scheduleMonitorTask(task)
	} else {
		// Stop the scheduler
		monitorLock.Lock()
		defer monitorLock.Unlock()
		if c, exists := monitorTaskSchedulers[taskId]; exists {
			c.Stop()
			delete(monitorTaskSchedulers, taskId)
		}
		return task.Update()
	}
}

// WebhookPayload Webhook payload structure
type WebhookPayload struct {
	Event        string      `json:"event"`
	TaskId       int         `json:"task_id"`
	TaskName     string      `json:"task_name"`
	ChannelId    int         `json:"channel_id"`
	Model        string      `json:"model"`
	Status       string      `json:"status"`
	ErrorMsg     string      `json:"error_message"`
	ResponseTime int         `json:"response_time"`
	Timestamp    string      `json:"timestamp"`
	AlertType    int         `json:"alert_type"`
}
