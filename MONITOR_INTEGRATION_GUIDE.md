# 监控工具集成指南

## 概述

本指南说明如何将新的监控工具集成到现有的New API程序中。**所有新代码都是独立的，不修改现有文件**。

## 集成步骤

### 第一步：查看新增文件

新增的文件位置：

```
model/monitor.go                 # 数据模型（约700行）
service/monitor.go              # 业务逻辑（约400行）
service/notification.go         # Webhook通知（约200行）
controller/monitor.go           # API控制器（约500行）
router/monitor.go               # 路由配置（约50行）
```

### 第二步：修改 main.go

在 `main.go` 中添加以下代码片段。**不要直接修改代码，请按照以下模板添加**。

#### A. 在初始化部分添加（在路由配置之后）

找到类似这样的代码段：
```go
// 在路由注册的地方，通常在 main.go 后期
func initRouter() {
    engine := gin.New()
    // ... existing routes ...

    // 添加以下代码：
    router.RegisterMonitorRoutes(engine)  // 注册监控路由
}
```

#### B. 在程序启动部分添加（在Gin服务器启动之后）

找到启动Gin引擎的地方，添加：
```go
func main() {
    // ... existing code ...

    // 初始化监控系统（在启动HTTP服务器之前）
    if err := service.InitMonitorSystem(); err != nil {
        log.Printf("Warning: Failed to initialize monitor system: %v", err)
        // 这个错误不应该阻止程序启动，仅记录警告
    }

    // 启动HTTP服务器
    // ... existing code ...
}
```

#### C. 在程序关闭部分添加（Graceful shutdown）

找到程序关闭的处理代码，添加：
```go
func main() {
    // ... existing code ...

    // 处理系统信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan

        // 优雅关闭监控系统
        service.StopMonitorSystem()

        // ... existing shutdown code ...
    }()
}
```

### 第三步：初始化数据库表

监控系统会在启动时自动创建所需的数据库表。表包括：
- `monitor_tasks` - 监控任务配置
- `monitor_results` - 监控结果记录
- `monitor_alerts` - 告警规则
- `monitor_webhooks` - Webhook配置

**无需手动执行迁移脚本**，程序启动时会自动检查并创建。

### 第四步：环境变量配置（可选）

在 `.env` 文件中添加以下配置（全部可选，有默认值）：

```bash
# 监控功能开关
MONITOR_ENABLED=true

# 是否在启动时自动初始化数据库表
MONITOR_DB_AUTO_INIT=true

# Webhook请求超时时间（秒）
MONITOR_WEBHOOK_TIMEOUT=10

# Webhook重试次数
MONITOR_WEBHOOK_RETRY=3

# 最大并发监控任务数
MONITOR_MAX_CONCURRENT_TASKS=5
```

## 完整的集成代码示例

### main.go 的修改示例

```go
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func main() {
	// ... 现有的初始化代码 ...

	engine := gin.Default()

	// ... 现有的路由注册 ...
	router.RegisterChannelRoutes(engine)
	router.RegisterUserRoutes(engine)
	// ... 其他现有路由 ...

	// 新增：注册监控工具路由
	router.RegisterMonitorRoutes(engine)

	// 新增：初始化监控系统
	if err := service.InitMonitorSystem(); err != nil {
		log.Printf("Warning: Failed to initialize monitor system: %v", err)
	}

	// 处理系统信号以实现优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan

		// 新增：优雅关闭监控系统
		service.StopMonitorSystem()

		// ... 现有的关闭代码 ...
		os.Exit(0)
	}()

	// 启动HTTP服务器
	if err := engine.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
```

## API 使用示例

### 创建监控任务

```bash
curl -X POST http://localhost:3000/api/monitor/tasks \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI GPT-4 监控",
    "enabled": true,
    "channels": [1, 2],
    "models": ["gpt-4", "gpt-4-turbo"],
    "schedule": {
      "interval": 300
    },
    "test_content": "你好，请告诉我你是什么模型？",
    "expected_pattern": "gpt-4.*",
    "webhook_urls": ["https://example.com/webhook"],
    "max_retries": 2,
    "timeout": 30,
    "remark": "监控主要的GPT-4渠道"
  }'
```

### 手动运行监控任务

```bash
curl -X POST http://localhost:3000/api/monitor/tasks/1/run-now \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### 获取监控结果

```bash
curl -X GET "http://localhost:3000/api/monitor/tasks/1/latest-results" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

### 配置告警

```bash
curl -X POST http://localhost:3000/api/monitor/tasks/1/alerts \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "alert_type": 0,
    "threshold": 3,
    "window_size": 60,
    "webhook_enabled": true,
    "page_alert_enabled": true
  }'
```

## 集成验证

集成完成后，验证步骤：

1. **检查程序启动日志**
   ```
   Monitor system initialized successfully
   ```

2. **检查数据库表**
   ```sql
   SHOW TABLES LIKE 'monitor_%';
   ```
   应该看到4张新表

3. **测试API**
   ```bash
   curl http://localhost:3000/api/monitor/tasks \
     -H "Authorization: Bearer YOUR_TOKEN"
   ```
   应该返回成功响应和空的任务列表

## 故障排除

### 问题1：监控系统初始化失败

**症状**：程序启动时看到 "Warning: Failed to initialize monitor system"

**原因**：
- 数据库连接失败
- 权限问题
- Go依赖未正确安装

**解决**：
```bash
# 检查数据库连接
# 检查日志中的具体错误消息
# 重新运行 go mod tidy
go mod tidy
```

### 问题2：路由未注册

**症状**：访问 `/api/monitor/tasks` 返回404

**原因**：未在 main.go 中调用 `router.RegisterMonitorRoutes()`

**解决**：
确保在 main.go 中添加了 `router.RegisterMonitorRoutes(engine)`

### 问题3：权限错误

**症状**：API调用返回401或403

**原因**：
- 未提供有效的admin token
- 用户权限不足

**解决**：
确保使用有效的admin用户令牌进行API调用

## 升级和与上游代码合并

### 优点

监控工具的设计保证了易于合并：

1. **完全独立的文件** - 新增4个新文件，不修改任何现有文件
2. **最小的集成点** - 只需在 main.go 中添加3-4行代码
3. **向后兼容** - 不改变任何现有的API或行为

### 合并步骤

当从上游更新代码时：

1. 拉取上游更新
   ```bash
   git pull upstream main
   ```

2. 如果 main.go 有冲突，手动添加我们的监控初始化代码
   ```go
   // 在合适的地方添加
   router.RegisterMonitorRoutes(engine)
   if err := service.InitMonitorSystem(); err != nil {
       log.Printf("Warning: %v", err)
   }
   ```

3. 其他新增文件不会产生冲突

## 前端集成

前端集成将在单独的指南中说明，包括：
- 添加监控菜单项
- 创建监控页面
- 实现API调用

## 常见问题

**Q: 监控任务什么时候开始执行？**
A: 在 `service.InitMonitorSystem()` 调用后，所有启用的监控任务会立即被调度。定时任务会根据配置的间隔执行。

**Q: 如果程序重启，监控任务会继续吗？**
A: 是的。程序启动时会从数据库读取所有启用的监控任务并重新调度。

**Q: 能否禁用监控功能但保留代码？**
A: 可以。在 `service.InitMonitorSystem()` 之前检查环境变量 `MONITOR_ENABLED`，如果为false则跳过初始化。

**Q: 监控会占用多少资源？**
A: 取决于配置的任务数量和执行频率。建议使用 `MONITOR_MAX_CONCURRENT_TASKS` 限制并发任务数。

## 支持和文档

- 详细的设计文档：`MONITOR_TOOL_DESIGN.md`
- 开发进度：`MONITOR_DEVELOPMENT_PROGRESS.md`
- API完整文档：在设计文档的"API接口设计"部分

