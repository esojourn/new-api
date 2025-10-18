package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestWebhookPayloadStructure tests the WebhookPayload structure
func TestWebhookPayloadStructure(t *testing.T) {
	payload := &WebhookPayload{
		Event:        "monitor_failure",
		TaskId:       1,
		TaskName:     "Test Task",
		ChannelId:    1,
		Model:        "gpt-4",
		Status:       "failed",
		ErrorMsg:     "API Error",
		ResponseTime: 5000,
		Timestamp:    time.Now().Format(time.RFC3339),
		AlertType:    0,
	}

	// Test that payload can be marshaled to JSON
	jsonBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	// Test that JSON can be unmarshaled back
	var unmarshaledPayload WebhookPayload
	err = json.Unmarshal(jsonBytes, &unmarshaledPayload)
	assert.NoError(t, err)

	assert.Equal(t, "monitor_failure", unmarshaledPayload.Event)
	assert.Equal(t, 1, unmarshaledPayload.TaskId)
	assert.Equal(t, "gpt-4", unmarshaledPayload.Model)
	assert.Equal(t, "failed", unmarshaledPayload.Status)
}

// TestWebhookPayloadFields tests individual webhook payload fields
func TestWebhookPayloadFields(t *testing.T) {
	payload := WebhookPayload{
		Event:        "monitor_success",
		TaskId:       42,
		TaskName:     "API Monitor",
		ChannelId:    10,
		Model:        "claude-3",
		Status:       "success",
		ErrorMsg:     "",
		ResponseTime: 1200,
		Timestamp:    "2025-10-18T10:00:00Z",
		AlertType:    1,
	}

	assert.Equal(t, "monitor_success", payload.Event)
	assert.Equal(t, 42, payload.TaskId)
	assert.Equal(t, "API Monitor", payload.TaskName)
	assert.Equal(t, 10, payload.ChannelId)
	assert.Equal(t, "claude-3", payload.Model)
	assert.Equal(t, "success", payload.Status)
	assert.Equal(t, "", payload.ErrorMsg)
	assert.Equal(t, 1200, payload.ResponseTime)
	assert.Equal(t, 1, payload.AlertType)
}

// TestMonitorServiceCreation tests that NotificationService can be created
func TestMonitorServiceCreation(t *testing.T) {
	notifService := NewNotificationService()
	assert.NotNil(t, notifService)
	assert.NotNil(t, notifService.httpClient)
}

// TestNotificationServiceHTTPClient tests the HTTP client configuration
func TestNotificationServiceHTTPClient(t *testing.T) {
	notifService := NewNotificationService()
	assert.NotNil(t, notifService.httpClient)

	// HTTP client should have a timeout configured
	assert.NotZero(t, notifService.httpClient.Timeout)
}

// TestScheduleConfigStructure tests the ScheduleConfig structure
func TestScheduleConfigStructure(t *testing.T) {
	// Test with interval
	config1 := map[string]interface{}{
		"interval": 900,
	}

	jsonBytes, _ := json.Marshal(config1)
	assert.NotNil(t, jsonBytes)

	// Test with daily schedule
	hour := 2
	minute := 30
	config2 := map[string]interface{}{
		"hour":   hour,
		"minute": minute,
	}

	jsonBytes, _ = json.Marshal(config2)
	assert.NotNil(t, jsonBytes)
}

// TestWebhookSignatureGeneration tests HMAC signature generation
func TestWebhookSignatureGeneration(t *testing.T) {
	data := []byte("test webhook data")
	secret := "test_secret"

	signature := generateHmacSignature(data, secret)

	// Signature should be a valid hex string
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64) // SHA256 produces 64 hex characters
}

// TestWebhookSignatureVerification tests signature verification
func TestWebhookSignatureVerification(t *testing.T) {
	data := []byte("test webhook data")
	secret := "test_secret"

	signature := generateHmacSignature(data, secret)

	// Correct signature should verify
	isValid := VerifyWebhookSignature(data, signature, secret)
	assert.True(t, isValid)

	// Wrong signature should not verify
	wrongSignature := "0000000000000000000000000000000000000000000000000000000000000000"
	isValid = VerifyWebhookSignature(data, wrongSignature, secret)
	assert.False(t, isValid)

	// Wrong secret should not verify
	isValid = VerifyWebhookSignature(data, signature, "wrong_secret")
	assert.False(t, isValid)
}

// TestDifferentWebhookSecrets tests that different secrets produce different signatures
func TestDifferentWebhookSecrets(t *testing.T) {
	data := []byte("webhook data")
	secret1 := "secret1"
	secret2 := "secret2"

	signature1 := generateHmacSignature(data, secret1)
	signature2 := generateHmacSignature(data, secret2)

	assert.NotEqual(t, signature1, signature2)
}

// TestEmptyWebhookSecret tests with empty secret
func TestEmptyWebhookSecret(t *testing.T) {
	data := []byte("test data")
	emptySecret := ""

	signature := generateHmacSignature(data, emptySecret)

	// Even with empty secret, signature should be generated
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64)

	// Verification should work with empty secret
	isValid := VerifyWebhookSignature(data, signature, emptySecret)
	assert.True(t, isValid)
}

// TestNotificationTemplateStructure tests the NotificationTemplate structure
func TestNotificationTemplateStructure(t *testing.T) {
	template := NotificationTemplate{
		Type:    "webhook",
		Format:  map[string]interface{}{},
		Webhook: "http://example.com/webhook",
		Secret:  "secret123",
	}

	assert.Equal(t, "webhook", template.Type)
	assert.Equal(t, "http://example.com/webhook", template.Webhook)
	assert.Equal(t, "secret123", template.Secret)
}

// TestMultipleTemplateTypes tests different notification template types
func TestMultipleTemplateTypes(t *testing.T) {
	types := []string{"webhook", "dingtalk", "slack", "custom"}

	for _, templateType := range types {
		template := NotificationTemplate{
			Type:    templateType,
			Webhook: "http://example.com/webhook",
		}
		assert.Equal(t, templateType, template.Type)
	}
}
