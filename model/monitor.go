package model

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MonitorTask 监控任务配置
type MonitorTask struct {
	Id               int            `json:"id" gorm:"primaryKey"`
	Name             string         `json:"name" gorm:"index;not null"`
	Enabled          bool           `json:"enabled" gorm:"default:true"`
	Channels         datatypes.JSON `json:"channels" gorm:"type:json"` // JSON array of channel IDs: [1,2,3]
	Models           datatypes.JSON `json:"models" gorm:"type:json"`   // JSON array of model names: ["gpt-4", "claude-3"]
	Schedule         datatypes.JSON `json:"schedule" gorm:"type:json"` // JSON object: {interval: 300, hour: null, minute: null}
	TestContent      string         `json:"test_content" gorm:"type:text;not null"`
	ExpectedPattern  string         `json:"expected_pattern" gorm:"type:text"` // Regex pattern for validation
	WebhookUrls      datatypes.JSON `json:"webhook_urls" gorm:"type:json"`     // JSON array of webhook URLs
	MaxRetries       int            `json:"max_retries" gorm:"default:2"`
	Timeout          int            `json:"timeout" gorm:"default:30"` // in seconds
	CreatedTime      int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64          `json:"updated_time" gorm:"bigint"`
	LastRunAt        int64          `json:"last_run_at" gorm:"bigint"` // timestamp of last run
	Remark           string         `json:"remark" gorm:"type:varchar(255)"`
	LastRunStatus    int            `json:"last_run_status" gorm:"default:0"` // 0=success, 1=failed, 2=running, -1=never run
	FailureCount     int            `json:"failure_count" gorm:"default:0"`    // consecutive failures
	SuccessCount     int            `json:"success_count" gorm:"default:0"`    // consecutive successes
	TotalRuns        int64          `json:"total_runs" gorm:"default:0"`
	TotalSuccesses   int64          `json:"total_successes" gorm:"default:0"`
	TotalFailures    int64          `json:"total_failures" gorm:"default:0"`
	AvgResponseTime  int            `json:"avg_response_time"` // in milliseconds
}

// MonitorResult 监控结果记录
type MonitorResult struct {
	Id              int64          `json:"id" gorm:"primaryKey"`
	TaskId          int            `json:"task_id" gorm:"index;not null"`
	ChannelId       int            `json:"channel_id" gorm:"index;not null"`
	Model           string         `json:"model" gorm:"index;not null"`
	Status          int            `json:"status" gorm:"default:0"`    // 0=success, 1=failed
	ResponseTime    int            `json:"response_time"`              // in milliseconds
	TokensUsed      int            `json:"tokens_used"`
	ErrorMessage    string         `json:"error_message" gorm:"type:text"`
	RequestId       string         `json:"request_id" gorm:"index"`
	FullResponse    string         `json:"full_response" gorm:"type:text"` // optional, complete response
	RetryCount      int            `json:"retry_count" gorm:"default:0"`
	WebhookNotified bool           `json:"webhook_notified" gorm:"default:false"`
	CreatedAt       int64          `json:"created_at" gorm:"bigint;index"`
	UpdatedAt       int64          `json:"updated_at" gorm:"bigint"`
}

// MonitorAlert 告警配置
type MonitorAlert struct {
	Id                int   `json:"id" gorm:"primaryKey"`
	TaskId            int   `json:"task_id" gorm:"index;not null"`
	AlertType         int   `json:"alert_type"` // 0=consecutive failures, 1=low success rate
	Threshold         int   `json:"threshold"` // e.g., 3 for 3 consecutive failures, or 90 for 90% success rate
	WindowSize        int   `json:"window_size"` // in minutes, time window for checking
	WebhookEnabled    bool  `json:"webhook_enabled" gorm:"default:false"`
	PageAlertEnabled  bool  `json:"page_alert_enabled" gorm:"default:false"`
	CreatedTime       int64 `json:"created_time" gorm:"bigint"`
	UpdatedTime       int64 `json:"updated_time" gorm:"bigint"`
	LastTriggeredTime int64 `json:"last_triggered_time" gorm:"bigint"`
}

// MonitorWebhook Webhook配置
type MonitorWebhook struct {
	Id       int    `json:"id" gorm:"primaryKey"`
	TaskId   int    `json:"task_id" gorm:"index;not null"`
	Url      string `json:"url" gorm:"type:text;not null"`
	Events   string `json:"events" gorm:"type:text"` // JSON array: ["failure", "success"]
	Secret   string `json:"secret" gorm:"type:varchar(255)"`
	Enabled  bool   `json:"enabled" gorm:"default:true"`
	LastTest int64  `json:"last_test" gorm:"bigint"`
}

// ScheduleConfig 时间表配置结构
type ScheduleConfig struct {
	Interval int    `json:"interval"` // 固定间隔（秒）
	Hour     *int   `json:"hour"`     // 小时（可选，用于每日定时）
	Minute   *int   `json:"minute"`   // 分钟（可选，用于每日定时）
	CronExpr string `json:"cron_expr"` // Cron表达式（可选）
}

// GetChannelIds 获取监控的渠道ID列表
func (task *MonitorTask) GetChannelIds() ([]int, error) {
	var ids []int
	if len(task.Channels) == 0 {
		return ids, nil
	}
	err := json.Unmarshal(task.Channels, &ids)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to unmarshal channel ids: task_id=%d, error=%v", task.Id, err))
		return nil, err
	}
	return ids, nil
}

// SetChannelIds 设置监控的渠道ID列表
func (task *MonitorTask) SetChannelIds(ids []int) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	task.Channels = data
	return nil
}

// GetModels 获取监控的模型列表
func (task *MonitorTask) GetModels() ([]string, error) {
	var models []string
	if len(task.Models) == 0 {
		return models, nil
	}
	err := json.Unmarshal(task.Models, &models)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to unmarshal models: task_id=%d, error=%v", task.Id, err))
		return nil, err
	}
	return models, nil
}

// SetModels 设置监控的模型列表
func (task *MonitorTask) SetModels(models []string) error {
	data, err := json.Marshal(models)
	if err != nil {
		return err
	}
	task.Models = data
	return nil
}

// GetSchedule 获取时间表配置
func (task *MonitorTask) GetSchedule() (ScheduleConfig, error) {
	schedule := ScheduleConfig{}
	if len(task.Schedule) == 0 {
		schedule.Interval = 300 // default 5 minutes
		return schedule, nil
	}
	err := json.Unmarshal(task.Schedule, &schedule)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to unmarshal schedule: task_id=%d, error=%v", task.Id, err))
		return schedule, err
	}
	return schedule, nil
}

// SetSchedule 设置时间表配置
func (task *MonitorTask) SetSchedule(schedule ScheduleConfig) error {
	data, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	task.Schedule = data
	return nil
}

// GetWebhookUrls 获取Webhook URL列表
func (task *MonitorTask) GetWebhookUrls() ([]string, error) {
	var urls []string
	if len(task.WebhookUrls) == 0 {
		return urls, nil
	}
	err := json.Unmarshal(task.WebhookUrls, &urls)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to unmarshal webhook urls: task_id=%d, error=%v", task.Id, err))
		return nil, err
	}
	return urls, nil
}

// SetWebhookUrls 设置Webhook URL列表
func (task *MonitorTask) SetWebhookUrls(urls []string) error {
	data, err := json.Marshal(urls)
	if err != nil {
		return err
	}
	task.WebhookUrls = data
	return nil
}

// Save 保存监控任务
func (task *MonitorTask) Save() error {
	now := common.GetTimestamp()
	if task.Id == 0 {
		task.CreatedTime = now
	}
	task.UpdatedTime = now
	return DB.Save(task).Error
}

// Update 更新监控任务
func (task *MonitorTask) Update() error {
	task.UpdatedTime = common.GetTimestamp()
	return DB.Model(task).Updates(task).Error
}

// Delete 删除监控任务
func (task *MonitorTask) Delete() error {
	// 级联删除相关数据
	if err := DB.Where("task_id = ?", task.Id).Delete(&MonitorResult{}).Error; err != nil {
		return err
	}
	if err := DB.Where("task_id = ?", task.Id).Delete(&MonitorAlert{}).Error; err != nil {
		return err
	}
	if err := DB.Where("task_id = ?", task.Id).Delete(&MonitorWebhook{}).Error; err != nil {
		return err
	}
	return DB.Delete(task).Error
}

// GetById 根据ID获取监控任务
func GetMonitorTaskById(id int) (*MonitorTask, error) {
	task := &MonitorTask{}
	err := DB.First(task, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("monitor task not found")
		}
		return nil, err
	}
	return task, nil
}

// GetAll 获取所有监控任务
func GetAllMonitorTasks() ([]*MonitorTask, error) {
	var tasks []*MonitorTask
	err := DB.Order("id desc").Find(&tasks).Error
	return tasks, err
}

// GetEnabledTasks 获取所有启用的监控任务
func GetEnabledMonitorTasks() ([]*MonitorTask, error) {
	var tasks []*MonitorTask
	err := DB.Where("enabled = ?", true).Order("id desc").Find(&tasks).Error
	return tasks, err
}

// GetPaginatedMonitorTasks 分页获取监控任务
func GetPaginatedMonitorTasks(offset int, limit int) ([]*MonitorTask, int64, error) {
	var tasks []*MonitorTask
	var total int64

	// Get total count
	if err := DB.Model(&MonitorTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := DB.Order("id desc").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

// SaveMonitorResult 保存监控结果
func (result *MonitorResult) Save() error {
	now := common.GetTimestamp()
	if result.CreatedAt == 0 {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	return DB.Save(result).Error
}

// GetResultsByTaskId 获取任务的监控结果
func GetMonitorResultsByTaskId(taskId int, limit int) ([]*MonitorResult, error) {
	var results []*MonitorResult
	err := DB.Where("task_id = ?", taskId).Order("created_at desc").Limit(limit).Find(&results).Error
	return results, err
}

// GetResultsByTaskIdAndDateRange 获取时间范围内的监控结果
func GetMonitorResultsByTaskIdAndDateRange(taskId int, startTime int64, endTime int64, limit int, offset int) ([]*MonitorResult, int64, error) {
	var results []*MonitorResult
	var total int64

	query := DB.Where("task_id = ? AND created_at >= ? AND created_at <= ?", taskId, startTime, endTime)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&results).Error
	return results, total, err
}

// GetLatestResultsByTask 获取任务的最新一次各个渠道和模型的结果
func GetLatestMonitorResultsByTask(taskId int) ([]*MonitorResult, error) {
	var results []*MonitorResult
	err := DB.
		Where("task_id = ?", taskId).
		Order("created_at desc").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	// Keep only the latest result for each (channel_id, model) combination
	latestMap := make(map[string]*MonitorResult)
	for _, result := range results {
		key := fmt.Sprintf("%d_%s", result.ChannelId, result.Model)
		if _, exists := latestMap[key]; !exists {
			latestMap[key] = result
		}
	}

	latestResults := make([]*MonitorResult, 0, len(latestMap))
	for _, result := range latestMap {
		latestResults = append(latestResults, result)
	}

	return latestResults, nil
}

// GetTaskStatistics 获取任务的统计信息
type TaskStatistics struct {
	TotalRuns       int64   `json:"total_runs"`
	TotalSuccesses  int64   `json:"total_successes"`
	TotalFailures   int64   `json:"total_failures"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime int     `json:"avg_response_time"`
	MinResponseTime int     `json:"min_response_time"`
	MaxResponseTime int     `json:"max_response_time"`
}

func GetMonitorTaskStatistics(taskId int) (*TaskStatistics, error) {
	var results []*MonitorResult
	err := DB.Where("task_id = ?", taskId).Find(&results).Error
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &TaskStatistics{}, nil
	}

	stats := &TaskStatistics{
		TotalRuns: int64(len(results)),
	}

	var totalResponseTime int64
	minResponseTime := int(^uint(0) >> 1) // max int
	maxResponseTime := 0

	for _, result := range results {
		if result.Status == 0 {
			stats.TotalSuccesses++
		} else {
			stats.TotalFailures++
		}
		totalResponseTime += int64(result.ResponseTime)
		if result.ResponseTime < minResponseTime {
			minResponseTime = result.ResponseTime
		}
		if result.ResponseTime > maxResponseTime {
			maxResponseTime = result.ResponseTime
		}
	}

	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalRuns) * 100
		stats.AvgResponseTime = int(totalResponseTime / stats.TotalRuns)
	}

	if minResponseTime == int(^uint(0)>>1) {
		minResponseTime = 0
	}

	stats.MinResponseTime = minResponseTime
	stats.MaxResponseTime = maxResponseTime

	return stats, nil
}

// SaveMonitorAlert 保存告警配置
func (alert *MonitorAlert) Save() error {
	now := common.GetTimestamp()
	if alert.Id == 0 {
		alert.CreatedTime = now
	}
	alert.UpdatedTime = now
	return DB.Save(alert).Error
}

// GetAlertsByTaskId 获取任务的告警配置
func GetMonitorAlertsByTaskId(taskId int) ([]*MonitorAlert, error) {
	var alerts []*MonitorAlert
	err := DB.Where("task_id = ?", taskId).Find(&alerts).Error
	return alerts, err
}

// SaveMonitorWebhook 保存Webhook配置
func (webhook *MonitorWebhook) Save() error {
	return DB.Save(webhook).Error
}

// GetWebhooksByTaskId 获取任务的Webhook配置
func GetMonitorWebhooksByTaskId(taskId int) ([]*MonitorWebhook, error) {
	var webhooks []*MonitorWebhook
	err := DB.Where("task_id = ? AND enabled = ?", taskId, true).Find(&webhooks).Error
	return webhooks, err
}

// InitMonitorTables 初始化监控相关的数据库表
func InitMonitorTables() error {
	if !DB.Migrator().HasTable(&MonitorTask{}) {
		if err := DB.Migrator().CreateTable(&MonitorTask{}); err != nil {
			return err
		}
		// Create indexes
		DB.Migrator().CreateIndex(&MonitorTask{}, "name")
		DB.Migrator().CreateIndex(&MonitorTask{}, "enabled")
	}

	if !DB.Migrator().HasTable(&MonitorResult{}) {
		if err := DB.Migrator().CreateTable(&MonitorResult{}); err != nil {
			return err
		}
		// Create indexes
		DB.Migrator().CreateIndex(&MonitorResult{}, "task_id")
		DB.Migrator().CreateIndex(&MonitorResult{}, "channel_id")
		DB.Migrator().CreateIndex(&MonitorResult{}, "model")
		DB.Migrator().CreateIndex(&MonitorResult{}, "request_id")
		DB.Migrator().CreateIndex(&MonitorResult{}, "created_at")
	}

	if !DB.Migrator().HasTable(&MonitorAlert{}) {
		if err := DB.Migrator().CreateTable(&MonitorAlert{}); err != nil {
			return err
		}
		// Create indexes
		DB.Migrator().CreateIndex(&MonitorAlert{}, "task_id")
	}

	if !DB.Migrator().HasTable(&MonitorWebhook{}) {
		if err := DB.Migrator().CreateTable(&MonitorWebhook{}); err != nil {
			return err
		}
		// Create indexes
		DB.Migrator().CreateIndex(&MonitorWebhook{}, "task_id")
	}

	return nil
}

// CleanOldMonitorResults 清理N天前的监控结果记录
func CleanOldMonitorResults(days int) error {
	if days <= 0 {
		days = 7 // default 7 days
	}

	// Calculate cutoff time (N days ago)
	cutoffTime := common.GetTimestamp() - int64(days*24*60*60)

	// Delete old results
	result := DB.Where("created_at < ?", cutoffTime).Delete(&MonitorResult{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		common.SysLog(fmt.Sprintf("cleaned %d old monitor results (older than %d days)", result.RowsAffected, days))
	}

	return nil
}
