package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterMonitorRoutes 注册监控相关的路由
func RegisterMonitorRoutes(router *gin.Engine) {
	// 创建 API 路由组
	monitorGroup := router.Group("/api/monitor")

	// 所有监控相关接口都需要admin权限
	monitorGroup.Use(middleware.AdminAuth())

	// 监控任务管理
	{
		// POST /api/monitor/tasks - 创建监控任务
		monitorGroup.POST("/tasks", controller.CreateMonitorTask)

		// GET /api/monitor/tasks - 获取监控任务列表（分页）
		monitorGroup.GET("/tasks", controller.GetMonitorTasks)

		// GET /api/monitor/tasks/:id - 获取单个监控任务
		monitorGroup.GET("/tasks/:id", controller.GetMonitorTask)

		// PUT /api/monitor/tasks/:id - 更新监控任务
		monitorGroup.PUT("/tasks/:id", controller.UpdateMonitorTask)

		// DELETE /api/monitor/tasks/:id - 删除监控任务
		monitorGroup.DELETE("/tasks/:id", controller.DeleteMonitorTask)

		// PATCH /api/monitor/tasks/:id/toggle - 启用/禁用监控任务
		monitorGroup.PATCH("/tasks/:id/toggle", controller.ToggleMonitorTask)

		// POST /api/monitor/tasks/:id/run-now - 手动立即运行监控任务
		monitorGroup.POST("/tasks/:id/run-now", controller.RunMonitorTaskNow)
	}

	// 监控结果查询
	{
		// GET /api/monitor/tasks/:id/latest-results - 获取最新的监控结果（各渠道/模型组合的最新一次）
		monitorGroup.GET("/tasks/:id/latest-results", controller.GetMonitorTaskResults)

		// GET /api/monitor/tasks/:id/results - 获取监控结果历史（支持日期范围过滤和分页）
		monitorGroup.GET("/tasks/:id/results", controller.GetMonitorTaskResultsHistory)

		// GET /api/monitor/tasks/:id/statistics - 获取任务统计信息
		monitorGroup.GET("/tasks/:id/statistics", controller.GetMonitorTaskStatistics)
	}

	// 告警管理
	{
		// POST /api/monitor/tasks/:id/alerts - 创建告警规则
		monitorGroup.POST("/tasks/:id/alerts", controller.CreateMonitorAlert)

		// GET /api/monitor/tasks/:id/alerts - 获取告警规则列表
		monitorGroup.GET("/tasks/:id/alerts", controller.GetMonitorAlerts)
	}

	// Webhook管理
	{
		// POST /api/monitor/tasks/:id/webhooks - 添加Webhook
		monitorGroup.POST("/tasks/:id/webhooks", controller.CreateMonitorWebhook)

		// GET /api/monitor/tasks/:id/webhooks - 获取Webhook列表
		monitorGroup.GET("/tasks/:id/webhooks", controller.GetMonitorWebhooks)

		// POST /api/monitor/tasks/:id/webhooks/:wid/test - 测试Webhook
		monitorGroup.POST("/tasks/:id/webhooks/:wid/test", controller.TestMonitorWebhook)

		// DELETE /api/monitor/tasks/:id/webhooks/:wid - 删除Webhook
		monitorGroup.DELETE("/tasks/:id/webhooks/:wid", controller.DeleteMonitorWebhook)
	}
}
