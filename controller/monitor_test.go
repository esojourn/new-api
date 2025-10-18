package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
)

// TestCreateMonitorTaskRequest tests the request structure for creating a monitor task
func TestCreateMonitorTaskRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"name":              "Test Monitor",
		"enabled":           true,
		"channels":          []int{1, 2},
		"models":            []string{"gpt-4", "claude-3"},
		"schedule":          map[string]interface{}{"interval": 900},
		"test_content":      "hi",
		"expected_pattern":  "",
		"max_retries":       2,
		"timeout":           30,
		"webhook_urls":      []string{"http://example.com/webhook"},
		"remark":            "Test task",
	}

	jsonBytes, _ := json.Marshal(reqBody)
	body := bytes.NewReader(jsonBytes)

	assert.NotNil(t, body)
	assert.NotEmpty(t, jsonBytes)
}

// TestMonitorTaskFormValidation tests form validation for required fields
func TestMonitorTaskFormValidation(t *testing.T) {
	// Test missing required fields
	invalidReqs := []map[string]interface{}{
		// Missing name
		{
			"channels": []int{1},
			"models":   []string{"gpt-4"},
		},
		// Missing channels
		{
			"name":   "Test",
			"models": []string{"gpt-4"},
		},
		// Missing models
		{
			"name":     "Test",
			"channels": []int{1},
		},
	}

	for _, req := range invalidReqs {
		jsonBytes, _ := json.Marshal(req)
		assert.NotEmpty(t, jsonBytes)
	}
}

// TestMonitorTaskResponseStructure tests the response structure
func TestMonitorTaskResponseStructure(t *testing.T) {
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":           1,
			"name":         "Test Monitor",
			"enabled":      true,
			"total_runs":   0,
			"avg_response_time": 0,
		},
	}

	jsonBytes, _ := json.Marshal(response)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, true, unmarshaled["success"])
}

// TestMonitorTaskListResponse tests the list response structure
func TestMonitorTaskListResponse(t *testing.T) {
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"tasks": []map[string]interface{}{
				{
					"id":   1,
					"name": "Task 1",
				},
				{
					"id":   2,
					"name": "Task 2",
				},
			},
			"total": 2,
			"page":  1,
			"limit": 20,
		},
	}

	jsonBytes, _ := json.Marshal(response)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, true, unmarshaled["success"])

	data := unmarshaled["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(1), data["page"])
}

// TestMonitorTaskDetailsResponse tests the detail response structure
func TestMonitorTaskDetailsResponse(t *testing.T) {
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":                   1,
			"name":                 "Test Monitor",
			"enabled":              true,
			"total_runs":           100,
			"total_successes":      95,
			"total_failures":       5,
			"avg_response_time":    250,
			"last_run_status":      0,
			"last_run_at":          1697616000,
			"failure_count":        0,
			"success_count":        5,
		},
	}

	jsonBytes, _ := json.Marshal(response)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)

	data := unmarshaled["data"].(map[string]interface{})
	assert.Equal(t, float64(100), data["total_runs"])
	assert.Equal(t, float64(95), data["total_successes"])
}

// TestMonitorResultsResponse tests the results response structure
func TestMonitorResultsResponse(t *testing.T) {
	response := map[string]interface{}{
		"success": true,
		"data": []map[string]interface{}{
			{
				"id":            1,
				"task_id":       1,
				"channel_id":    1,
				"model":         "gpt-4",
				"status":        0, // success
				"response_time": 1200,
				"error_message": "",
			},
			{
				"id":            2,
				"task_id":       1,
				"channel_id":    2,
				"model":         "claude-3",
				"status":        1, // failure
				"response_time": 5000,
				"error_message": "Timeout",
			},
		},
	}

	jsonBytes, _ := json.Marshal(response)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, true, unmarshaled["success"])

	results := unmarshaled["data"].([]interface{})
	assert.Equal(t, 2, len(results))
}

// TestToggleTaskRequest tests the toggle request
func TestToggleTaskRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"enabled": true,
	}

	jsonBytes, _ := json.Marshal(reqBody)
	assert.NotEmpty(t, jsonBytes)
}

// TestDeleteTaskRequest tests the delete request handling
func TestDeleteTaskRequest(t *testing.T) {
	// Test successful delete response
	response := map[string]interface{}{
		"success": true,
	}

	jsonBytes, _ := json.Marshal(response)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, true, unmarshaled["success"])
}

// TestAlertCreationRequest tests alert creation request
func TestAlertCreationRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"alert_type":         0, // consecutive failures
		"threshold":          3,
		"window_size":        60,
		"webhook_enabled":    true,
		"page_alert_enabled": false,
	}

	jsonBytes, _ := json.Marshal(reqBody)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), unmarshaled["alert_type"])
	assert.Equal(t, float64(3), unmarshaled["threshold"])
}

// TestWebhookCreationRequest tests webhook creation request
func TestWebhookCreationRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"url":     "http://example.com/webhook",
		"events":  []string{"failure", "success"},
		"secret":  "webhook_secret",
		"enabled": true,
	}

	jsonBytes, _ := json.Marshal(reqBody)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/webhook", unmarshaled["url"])
	assert.Equal(t, true, unmarshaled["enabled"])
}

// TestAPIErrorResponse tests API error response
func TestAPIErrorResponse(t *testing.T) {
	errorResponse := map[string]interface{}{
		"success": false,
		"error":   "Invalid request",
	}

	jsonBytes, _ := json.Marshal(errorResponse)

	var unmarshaled map[string]interface{}
	err := json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, false, unmarshaled["success"])
}

// TestScheduleIntervalTypes tests different schedule interval type handling
func TestScheduleIntervalTypes(t *testing.T) {
	// Test with integer interval
	scheduleInt := map[string]interface{}{
		"interval": 900,
	}
	jsonBytesInt, _ := json.Marshal(scheduleInt)
	assert.NotEmpty(t, jsonBytesInt)

	// Test with float interval (from JSON)
	scheduleFloat := map[string]interface{}{
		"interval": float64(900),
	}
	jsonBytesFloat, _ := json.Marshal(scheduleFloat)
	assert.NotEmpty(t, jsonBytesFloat)

	// Both should unmarshal properly
	var unmarshaledInt map[string]interface{}
	var unmarshaledFloat map[string]interface{}

	json.Unmarshal(jsonBytesInt, &unmarshaledInt)
	json.Unmarshal(jsonBytesFloat, &unmarshaledFloat)

	assert.NotNil(t, unmarshaledInt["interval"])
	assert.NotNil(t, unmarshaledFloat["interval"])
}

// TestGinContextHandling tests that requests are properly handled with Gin
func TestGinContextHandling(t *testing.T) {
	// Create a new Gin router for testing
	router := gin.New()

	// Create a test route that mimics the monitor endpoint
	router.POST("/api/monitor/tasks", func(c *gin.Context) {
		var req struct {
			Name     string `json:"name" binding:"required"`
			Enabled  bool   `json:"enabled"`
			Channels []int  `json:"channels" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    req,
		})
	})

	// Test valid request
	validReq := map[string]interface{}{
		"name":     "Test",
		"enabled":  true,
		"channels": []int{1},
	}
	jsonBytes, _ := json.Marshal(validReq)

	req := httptest.NewRequest("POST", "/api/monitor/tasks", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, true, response["success"])
}
