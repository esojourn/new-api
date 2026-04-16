package middleware

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const maxLogBodySize = 10 * 1024 // 最大记录 10KB 的请求体，防止日志过大

var (
	requestLogEnabled bool
	requestLogDir     string
	requestLogFile    *os.File
	requestLogger     *log.Logger
	requestLogMu      sync.RWMutex
	requestLogCount   int
	requestLogMaxRows = 100000 // 单文件最大行数，超过后自动轮转
)

// InitRequestLogger 初始化请求体日志记录器
// logDir: 日志目录路径，为空则使用默认日志目录
func InitRequestLogger(logDir string) {
	envVal := os.Getenv("REQUEST_LOG_ENABLED")
	if envVal != "true" {
		return
	}
	requestLogEnabled = true

	if logDir == "" {
		logDir = "./logs"
	}
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0777); err != nil {
			common.SysError(fmt.Sprintf("failed to create request log dir: %v", err))
			requestLogEnabled = false
			return
		}
	}
	requestLogDir = logDir

	if err := rotateRequestLogFile(); err != nil {
		common.SysError(fmt.Sprintf("failed to init request log file: %v", err))
		requestLogEnabled = false
		return
	}

	common.SysLog(fmt.Sprintf("request body logger enabled, log dir: %s", logDir))
}

func rotateRequestLogFile() error {
	requestLogMu.Lock()
	defer requestLogMu.Unlock()

	if requestLogFile != nil {
		_ = requestLogFile.Close()
	}

	filename := fmt.Sprintf("request-%s.log", time.Now().Format("20060102-150405"))
	logPath := filepath.Join(requestLogDir, filename)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	requestLogFile = f
	requestLogger = log.New(f, "", 0) // 不使用 log 自带的前缀，自行格式化
	requestLogCount = 0
	return nil
}

// RequestBodyLogger 请求体日志中间件
// 通过环境变量 REQUEST_LOG_ENABLED=true 启用
// 将请求体内容写入独立的日志文件 (logs/request-*.log)
func RequestBodyLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requestLogEnabled {
			c.Next()
			return
		}

		// 先执行后续 handler，让请求体被 BodyStorage 缓存
		c.Next()

		logRequestBody(c)
	}
}

func logRequestBody(c *gin.Context) {
	storage, exists := c.Get(common.KeyBodyStorage)
	if !exists || storage == nil {
		return
	}

	bs, ok := storage.(common.BodyStorage)
	if !ok {
		return
	}

	bodyBytes, err := bs.Bytes()
	if err != nil {
		return
	}

	body := string(bodyBytes)
	if len(body) == 0 {
		return
	}

	// 截断过长的请求体
	truncated := false
	if len(body) > maxLogBodySize {
		body = body[:maxLogBodySize]
		truncated = true
	}

	// 构建日志行
	now := time.Now().Format("2006/01/02 - 15:04:05")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[REQ] %s | %s | %s %s", now, c.ClientIP(), c.Request.Method, c.Request.URL.Path))

	requestId := c.GetString(common.RequestIdKey)
	if requestId != "" {
		sb.WriteString(fmt.Sprintf(" | id=%s", requestId))
	}
	tokenName := c.GetString("token_name")
	if tokenName != "" {
		sb.WriteString(fmt.Sprintf(" | token=%s", tokenName))
	}
	username := c.GetString("username")
	if username != "" {
		sb.WriteString(fmt.Sprintf(" | user=%s", username))
	}
	model := c.GetString("original_model")
	if model != "" {
		sb.WriteString(fmt.Sprintf(" | model=%s", model))
	}

	sb.WriteString(fmt.Sprintf(" | size=%d", len(bodyBytes)))
	if truncated {
		sb.WriteString(" (truncated)")
	}
	sb.WriteString("\n")
	sb.WriteString(body)

	writeRequestLog(sb.String())
}

func writeRequestLog(msg string) {
	requestLogMu.RLock()
	l := requestLogger
	requestLogMu.RUnlock()

	if l == nil {
		return
	}

	l.Println(msg)

	requestLogMu.Lock()
	requestLogCount++
	needRotate := requestLogCount >= requestLogMaxRows
	requestLogMu.Unlock()

	if needRotate {
		if err := rotateRequestLogFile(); err != nil {
			common.SysError(fmt.Sprintf("failed to rotate request log: %v", err))
		}
	}
}
