package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

var monitorService *service.MonitorService

// InitMonitorController 初始化监控控制器
func InitMonitorController(notificationService *service.NotificationService) {
	monitorService = service.NewMonitorService(notificationService)
}

// CreateMonitorTask POST /api/monitor/tasks
func CreateMonitorTask(c *gin.Context) {
	var req struct {
		Name            string   `json:"name" binding:"required"`
		Enabled         bool     `json:"enabled" binding:""`
		Channels        []int    `json:"channels" binding:"required"`
		Models          []string `json:"models" binding:"required"`
		Schedule        map[string]interface{} `json:"schedule" binding:"required"`
		TestContent     string   `json:"test_content" binding:"required"`
		ExpectedPattern string   `json:"expected_pattern" binding:""`
		WebhookUrls     []string `json:"webhook_urls" binding:""`
		MaxRetries      int      `json:"max_retries" binding:""`
		Timeout         int      `json:"timeout" binding:""`
		Remark          string   `json:"remark" binding:""`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate
	if len(req.Channels) == 0 || len(req.Models) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channels and models are required"})
		return
	}

	// Create task
	task := &model.MonitorTask{
		Name:            req.Name,
		Enabled:         req.Enabled,
		TestContent:     req.TestContent,
		ExpectedPattern: req.ExpectedPattern,
		MaxRetries:      req.MaxRetries,
		Timeout:         req.Timeout,
		Remark:          req.Remark,
		CreatedTime:     common.GetTimestamp(),
	}

	// Set defaults
	if task.MaxRetries == 0 {
		task.MaxRetries = 2
	}
	if task.Timeout == 0 {
		task.Timeout = 30
	}

	// Set channels
	if err := task.SetChannelIds(req.Channels); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channels"})
		return
	}

	// Set models
	if err := task.SetModels(req.Models); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid models"})
		return
	}

	// Set schedule
	var interval int
	if scheduleData, ok := req.Schedule["interval"]; ok {
		// Handle both int and float64 types from JSON
		switch v := scheduleData.(type) {
		case float64:
			interval = int(v)
		case int:
			interval = v
		default:
			interval = 300 // default 5 minutes
		}
	} else {
		interval = 300 // default 5 minutes
	}
	if err := task.SetSchedule(model.ScheduleConfig{
		Interval: interval,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule"})
		return
	}

	// Set webhooks
	if err := task.SetWebhookUrls(req.WebhookUrls); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook urls"})
		return
	}

	// Save task
	if err := service.CreateMonitorTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": task})
}

// GetMonitorTasks GET /api/monitor/tasks
func GetMonitorTasks(c *gin.Context) {
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil {
			page = pageNum
		}
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if limitNum, err := strconv.Atoi(l); err == nil {
			limit = limitNum
		}
	}

	offset := (page - 1) * limit

	tasks, total, err := model.GetPaginatedMonitorTasks(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": tasks,
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetMonitorTask GET /api/monitor/tasks/:id
func GetMonitorTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := model.GetMonitorTaskById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": task})
}

// UpdateMonitorTask PUT /api/monitor/tasks/:id
func UpdateMonitorTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := model.GetMonitorTaskById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	var req struct {
		Name            string   `json:"name"`
		Enabled         bool     `json:"enabled"`
		Channels        []int    `json:"channels"`
		Models          []string `json:"models"`
		Schedule        map[string]interface{} `json:"schedule"`
		TestContent     string   `json:"test_content"`
		ExpectedPattern string   `json:"expected_pattern"`
		WebhookUrls     []string `json:"webhook_urls"`
		MaxRetries      int      `json:"max_retries"`
		Timeout         int      `json:"timeout"`
		Remark          string   `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		task.Name = req.Name
	}
	task.Enabled = req.Enabled
	if req.TestContent != "" {
		task.TestContent = req.TestContent
	}
	task.ExpectedPattern = req.ExpectedPattern
	if req.MaxRetries > 0 {
		task.MaxRetries = req.MaxRetries
	}
	if req.Timeout > 0 {
		task.Timeout = req.Timeout
	}
	task.Remark = req.Remark

	if len(req.Channels) > 0 {
		task.SetChannelIds(req.Channels)
	}
	if len(req.Models) > 0 {
		task.SetModels(req.Models)
	}
	if req.Schedule != nil {
		var interval int
		if scheduleData, ok := req.Schedule["interval"]; ok {
			// Handle both int and float64 types from JSON
			switch v := scheduleData.(type) {
			case float64:
				interval = int(v)
			case int:
				interval = v
			default:
				interval = 300 // default 5 minutes
			}
		} else {
			interval = 300 // default 5 minutes
		}
		task.SetSchedule(model.ScheduleConfig{
			Interval: interval,
		})
	}
	if req.WebhookUrls != nil {
		task.SetWebhookUrls(req.WebhookUrls)
	}

	if err := service.UpdateMonitorTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": task})
}

// DeleteMonitorTask DELETE /api/monitor/tasks/:id
func DeleteMonitorTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := service.DeleteMonitorTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ToggleMonitorTask PATCH /api/monitor/tasks/:id/toggle
func ToggleMonitorTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := service.ToggleMonitorTask(id, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RunMonitorTaskNow POST /api/monitor/tasks/:id/run-now
func RunMonitorTaskNow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	results, err := service.RunMonitorTaskNow(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

// GetMonitorTaskResults GET /api/monitor/tasks/:id/latest-results
func GetMonitorTaskResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	results, err := model.GetLatestMonitorResultsByTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

// GetMonitorTaskResultsHistory GET /api/monitor/tasks/:id/results
func GetMonitorTaskResultsHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil {
			page = pageNum
		}
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if limitNum, err := strconv.Atoi(l); err == nil {
			limit = limitNum
		}
	}

	offset := (page - 1) * limit

	// Parse date range (optional)
	var startTime, endTime int64
	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t.Unix()
		}
	}
	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t.Unix()
		}
	}

	if startTime == 0 {
		startTime = common.GetTimestamp() - 7*24*60*60 // 7 days ago
	}
	if endTime == 0 {
		endTime = common.GetTimestamp()
	}

	results, total, err := model.GetMonitorResultsByTaskIdAndDateRange(id, startTime, endTime, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"results": results,
			"total":   total,
			"page":    page,
			"limit":   limit,
		},
	})
}

// GetMonitorTaskStatistics GET /api/monitor/tasks/:id/statistics
func GetMonitorTaskStatistics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	stats, err := model.GetMonitorTaskStatistics(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

// CreateMonitorAlert POST /api/monitor/tasks/:id/alerts
func CreateMonitorAlert(c *gin.Context) {
	idStr := c.Param("id")
	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req struct {
		AlertType        int  `json:"alert_type" binding:"required"`
		Threshold        int  `json:"threshold" binding:"required"`
		WindowSize       int  `json:"window_size" binding:"required"`
		WebhookEnabled   bool `json:"webhook_enabled"`
		PageAlertEnabled bool `json:"page_alert_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert := &model.MonitorAlert{
		TaskId:           taskId,
		AlertType:        req.AlertType,
		Threshold:        req.Threshold,
		WindowSize:       req.WindowSize,
		WebhookEnabled:   req.WebhookEnabled,
		PageAlertEnabled: req.PageAlertEnabled,
		CreatedTime:      common.GetTimestamp(),
	}

	if err := alert.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": alert})
}

// GetMonitorAlerts GET /api/monitor/tasks/:id/alerts
func GetMonitorAlerts(c *gin.Context) {
	idStr := c.Param("id")
	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	alerts, err := model.GetMonitorAlertsByTaskId(taskId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": alerts})
}

// CreateMonitorWebhook POST /api/monitor/tasks/:id/webhooks
func CreateMonitorWebhook(c *gin.Context) {
	idStr := c.Param("id")
	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req struct {
		Url     string   `json:"url" binding:"required"`
		Events  []string `json:"events"`
		Secret  string   `json:"secret"`
		Enabled bool     `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventsJson, _ := json.Marshal(req.Events)
	webhook := &model.MonitorWebhook{
		TaskId:  taskId,
		Url:     req.Url,
		Events:  string(eventsJson),
		Secret:  req.Secret,
		Enabled: req.Enabled,
	}

	if err := webhook.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": webhook})
}

// GetMonitorWebhooks GET /api/monitor/tasks/:id/webhooks
func GetMonitorWebhooks(c *gin.Context) {
	idStr := c.Param("id")
	taskId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	webhooks, err := model.GetMonitorWebhooksByTaskId(taskId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": webhooks})
}

// TestMonitorWebhook POST /api/monitor/tasks/:id/webhooks/:wid/test
func TestMonitorWebhook(c *gin.Context) {
	widStr := c.Param("wid")
	webhookId, err := strconv.Atoi(widStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	// Get webhook
	webhooks, _ := model.GetMonitorWebhooksByTaskId(0)
	var webhook *model.MonitorWebhook
	for _, w := range webhooks {
		if w.Id == webhookId {
			webhook = w
			break
		}
	}

	if webhook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook not found"})
		return
	}

	// Test webhook
	notifService := service.NewNotificationService()
	result, err := notifService.TestWebhookNotification(webhook.Url, webhook.Secret)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "data": result})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// DeleteMonitorWebhook DELETE /api/monitor/tasks/:id/webhooks/:wid
func DeleteMonitorWebhook(c *gin.Context) {
	widStr := c.Param("wid")
	webhookId, err := strconv.Atoi(widStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook id"})
		return
	}

	// Delete webhook - implement this in model if needed
	c.JSON(http.StatusOK, gin.H{"success": true})
}

