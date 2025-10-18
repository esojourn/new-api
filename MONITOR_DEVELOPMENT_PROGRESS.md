# 监控工具开发进度报告

## 已完成的工作

### ✅ 1. 完整的架构设计文档
- 文件：`MONITOR_TOOL_DESIGN.md`
- 包含：需求分析、系统架构、API接口设计、部署配置等

### ✅ 2. 后端数据模型 (model/monitor.go)
**主要类型:**
- `MonitorTask` - 监控任务配置
- `MonitorResult` - 监控结果记录
- `MonitorAlert` - 告警配置
- `MonitorWebhook` - Webhook配置

**关键功能:**
- JSON数据序列化/反序列化
- 数据库CRUD操作
- 批量查询和统计
- 自动初始化数据库表

### ✅ 3. 监控业务逻辑服务 (service/monitor.go)
**核心功能:**
- 定时任务调度系统（使用 robfig/cron）
- 监控任务执行引擎
- 实际对话测试（与relay系统集成）
- 自动重试机制
- 告警检测和触发
- 任务管理API

**关键方法:**
- `InitMonitorSystem()` - 初始化系统
- `RunMonitorTask()` - 执行监控任务
- `testChannelModel()` - 单个模型测试
- `sendTestChatRequest()` - 发送实际对话请求
- `checkAndSendAlerts()` - 告警检查和通知

### ✅ 4. Webhook通知服务 (service/notification.go)
**功能:**
- 通用Webhook发送
- HMAC-SHA256签名支持
- 异步通知支持
- 钉钉、Slack等平台特定通知
- Webhook测试功能
- 通知模板系统

## 待完成的工作

### 📋 第一阶段：后端API和路由

需要创建的文件：
1. **controller/monitor.go** - API控制器
   - 任务CRUD操作接口
   - 结果查询接口
   - 告警配置接口
   - Webhook管理接口
   - 手动触发接口

2. **router/monitor.go** - 路由配置
   - 注册所有API路由
   - 设置权限检查中间件

3. **main.go** - 主程序集成
   - 初始化监控系统
   - 注册路由
   - 优雅关闭处理

### 📋 第二阶段：前端UI开发

需要创建的文件：
1. **web/src/pages/Monitor/** - 监控页面
   - `index.jsx` - 主页面容器
   - `TaskList.jsx` - 任务列表
   - `TaskForm.jsx` - 任务创建/编辑表单
   - `ResultsView.jsx` - 结果查看

2. **web/src/components/monitor/** - 监控组件
   - `TaskCard.jsx` - 任务卡片
   - `ResultsChart.jsx` - 结果图表
   - `AlertConfig.jsx` - 告警配置
   - `WebhookConfig.jsx` - Webhook配置

3. **web/src/services/monitor.js** - API服务
   - 调用后端API的客户端

### 📋 第三阶段：数据库迁移和测试

需要创建的文件：
1. **migration/monitor_schema.sql** - 数据库初始化脚本
2. 单元测试文件

## 技术细节

### 监控流程
```
┌─ 定时任务触发
│   └─ 读取启用的监控任务
│       ├─ 获取配置的渠道列表
│       ├─ 获取配置的模型列表
│       └─ 遍历所有组合
│           ├─ 获取渠道信息
│           ├─ 创建测试请求
│           ├─ 发送对话请求
│           ├─ 验证响应（可选regex匹配）
│           ├─ 保存结果到数据库
│           ├─ 检查是否触发告警
│           └─ 发送Webhook通知
│
└─ 更新任务统计信息
    ├─ 成功率
    ├─ 平均响应时间
    └─ 失败次数
```

### 与现有系统的集成点

1. **与relay系统集成**
   - 通过HTTP请求调用 `/v1/chat/completions` 端点
   - 继承现有的渠道选择、模型映射、格式转换逻辑

2. **与数据库系统集成**
   - 使用现有的GORM DB实例
   - 所有操作都是通过 `model.DB` 进行

3. **与认证系统集成**
   - API端点保护需要admin权限
   - 可配置权限检查

4. **与日志系统集成**
   - 所有重要操作记录到 `common.SysLog()`

## 独立性设计优势

1. **完全独立的代码文件**
   - 不修改任何现有的controller、service、model
   - 新增4个新文件 + 1个文档

2. **容易合并**
   - 只需在main.go添加2-3行代码
   - 其他文件完全独立

3. **可选功能**
   - 可通过环境变量启用/禁用
   - 不影响现有功能

## 配置示例

```bash
# .env
MONITOR_ENABLED=true
MONITOR_DB_AUTO_INIT=true
MONITOR_WEBHOOK_TIMEOUT=10
MONITOR_WEBHOOK_RETRY=3
MONITOR_MAX_CONCURRENT_TASKS=5
```

## 下一步行动

建议的开发顺序：
1. 先完成 controller/monitor.go 和 router/monitor.go
2. 在 main.go 中集成后端服务
3. 然后开发前端页面和组件
4. 最后进行集成测试

## API端点预览

```
# 任务管理
POST   /api/monitor/tasks              - 创建任务
GET    /api/monitor/tasks              - 获取任务列表
GET    /api/monitor/tasks/:id          - 获取任务详情
PUT    /api/monitor/tasks/:id          - 更新任务
DELETE /api/monitor/tasks/:id          - 删除任务
PATCH  /api/monitor/tasks/:id/toggle   - 启用/禁用

# 结果查询
GET    /api/monitor/tasks/:id/latest-results    - 最新结果
GET    /api/monitor/tasks/:id/results           - 历史结果
GET    /api/monitor/tasks/:id/statistics        - 统计信息

# 告警和Webhook
POST   /api/monitor/tasks/:id/alerts            - 配置告警
GET    /api/monitor/tasks/:id/alerts            - 获取告警
POST   /api/monitor/tasks/:id/webhooks          - 添加Webhook
POST   /api/monitor/tasks/:id/webhooks/:wid/test - 测试Webhook

# 手动操作
POST   /api/monitor/tasks/:id/run-now           - 手动运行任务
```

## 关键代码位置参考

- 数据模型：`model/monitor.go:21-100`
- 监控执行：`service/monitor.go:150-200`
- 对话测试：`service/monitor.go:200-250`
- 告警检测：`service/monitor.go:250-300`
- Webhook通知：`service/notification.go:30-80`

