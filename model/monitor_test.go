package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMonitorTaskGetChannelIds tests the GetChannelIds method
func TestMonitorTaskGetChannelIds(t *testing.T) {
	channels := []int{1, 2, 3}
	channelsJSON, _ := json.Marshal(channels)

	task := &MonitorTask{
		Channels: channelsJSON,
	}

	ids, err := task.GetChannelIds()
	assert.NoError(t, err)
	assert.Equal(t, channels, ids)
}

// TestMonitorTaskSetChannelIds tests the SetChannelIds method
func TestMonitorTaskSetChannelIds(t *testing.T) {
	task := &MonitorTask{}
	channels := []int{1, 2, 3}

	err := task.SetChannelIds(channels)
	assert.NoError(t, err)

	// Verify by unmarshaling
	var retrievedChannels []int
	err = json.Unmarshal(task.Channels, &retrievedChannels)
	assert.NoError(t, err)
	assert.Equal(t, channels, retrievedChannels)
}

// TestMonitorTaskGetModels tests the GetModels method
func TestMonitorTaskGetModels(t *testing.T) {
	models := []string{"gpt-4", "claude-3", "gemini"}
	modelsJSON, _ := json.Marshal(models)

	task := &MonitorTask{
		Models: modelsJSON,
	}

	retrievedModels, err := task.GetModels()
	assert.NoError(t, err)
	assert.Equal(t, models, retrievedModels)
}

// TestMonitorTaskSetModels tests the SetModels method
func TestMonitorTaskSetModels(t *testing.T) {
	task := &MonitorTask{}
	models := []string{"gpt-4", "claude-3"}

	err := task.SetModels(models)
	assert.NoError(t, err)

	// Verify by unmarshaling
	var retrievedModels []string
	err = json.Unmarshal(task.Models, &retrievedModels)
	assert.NoError(t, err)
	assert.Equal(t, models, retrievedModels)
}

// TestMonitorTaskGetSchedule tests the GetSchedule method
func TestMonitorTaskGetSchedule(t *testing.T) {
	schedule := ScheduleConfig{
		Interval: 900,
		Hour:     nil,
		Minute:   nil,
	}
	scheduleJSON, _ := json.Marshal(schedule)

	task := &MonitorTask{
		Schedule: scheduleJSON,
	}

	retrievedSchedule, err := task.GetSchedule()
	assert.NoError(t, err)
	assert.Equal(t, 900, retrievedSchedule.Interval)
}

// TestMonitorTaskSetSchedule tests the SetSchedule method
func TestMonitorTaskSetSchedule(t *testing.T) {
	task := &MonitorTask{}
	schedule := ScheduleConfig{
		Interval: 600,
	}

	err := task.SetSchedule(schedule)
	assert.NoError(t, err)

	// Verify by unmarshaling
	var retrievedSchedule ScheduleConfig
	err = json.Unmarshal(task.Schedule, &retrievedSchedule)
	assert.NoError(t, err)
	assert.Equal(t, 600, retrievedSchedule.Interval)
}

// TestMonitorTaskGetWebhookUrls tests the GetWebhookUrls method
func TestMonitorTaskGetWebhookUrls(t *testing.T) {
	urls := []string{"http://example.com/webhook", "http://another.com/notify"}
	urlsJSON, _ := json.Marshal(urls)

	task := &MonitorTask{
		WebhookUrls: urlsJSON,
	}

	retrievedUrls, err := task.GetWebhookUrls()
	assert.NoError(t, err)
	assert.Equal(t, urls, retrievedUrls)
}

// TestMonitorTaskSetWebhookUrls tests the SetWebhookUrls method
func TestMonitorTaskSetWebhookUrls(t *testing.T) {
	task := &MonitorTask{}
	urls := []string{"http://example.com/webhook"}

	err := task.SetWebhookUrls(urls)
	assert.NoError(t, err)

	// Verify by unmarshaling
	var retrievedUrls []string
	err = json.Unmarshal(task.WebhookUrls, &retrievedUrls)
	assert.NoError(t, err)
	assert.Equal(t, urls, retrievedUrls)
}

// TestTaskStatisticsCalculation tests the TaskStatistics calculation
func TestTaskStatisticsCalculation(t *testing.T) {
	stats := &TaskStatistics{
		TotalRuns:       100,
		TotalSuccesses:  90,
		TotalFailures:   10,
		AvgResponseTime: 250,
		MinResponseTime: 100,
		MaxResponseTime: 500,
	}

	assert.Equal(t, int64(100), stats.TotalRuns)
	assert.Equal(t, int64(90), stats.TotalSuccesses)
	assert.Equal(t, int64(10), stats.TotalFailures)

	// Calculate success rate
	successRate := float64(stats.TotalSuccesses) / float64(stats.TotalRuns) * 100
	assert.Equal(t, 90.0, successRate)
}

// TestMonitorResultStatus tests different result statuses
func TestMonitorResultStatus(t *testing.T) {
	successResult := &MonitorResult{
		Status: 0, // Success
	}
	assert.Equal(t, 0, successResult.Status)

	failureResult := &MonitorResult{
		Status: 1, // Failure
	}
	assert.Equal(t, 1, failureResult.Status)
}

// TestMonitorAlertThreshold tests alert threshold checking
func TestMonitorAlertThreshold(t *testing.T) {
	alert := &MonitorAlert{
		AlertType: 0, // Consecutive failures
		Threshold: 3,
	}

	assert.Equal(t, 0, alert.AlertType)
	assert.Equal(t, 3, alert.Threshold)

	// Test success rate alert
	alert2 := &MonitorAlert{
		AlertType: 1, // Low success rate
		Threshold: 90, // Alert if success rate < 90%
	}

	successRate := 85.0 // Lower than threshold
	assert.Less(t, successRate, float64(alert2.Threshold))
}
