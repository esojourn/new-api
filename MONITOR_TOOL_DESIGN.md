# 监控测试工具（Monitor Tool）- 完整设计文档

## 1. 需求分析

### 问题定义
- 现有的渠道测试功能只进行简单的连接测试
- 实际使用时可能出现测试通过但对话失败的情况
- 需要长期、定时监控渠道和模型的实际可用性

### 核心需求
1. **多维度监控**
   - 支持监控指定的一个或多个渠道
   - 支持监控指定的一个或多个模型
   - 可组合选择渠道+模型的监控组合

2. **定时实际对话测试**
   - 定时发送真实的对话请求（不只是连接测试）
   - 记录详细的测试结果（成功率、响应时间、token消耗等）
   - 支持多个测试时间间隔配置

3. **告警和通知**
   - 监控失败时显示页面提示
   - 通过Webhook通知第三方工具（钉钉、Slack等）
   - 支持多个Webhook配置

4. **低侵入性**
   - 所有新代码放在独立文件中
   - 不修改现有代码
   - 易于与上游代码合并

## 2. 系统架构

### 2.1 数据库表结构

```
monitor_tasks (监控任务)
├── id
├── name              (任务名称)
├── enabled           (是否启用)
├── channels          (监控的渠道ID列表，JSON数组)
├── models            (监控的模型列表，JSON数组)
├── schedule          (时间表配置，JSON)
│   ├── interval      (间隔时间，秒)
│   ├── hour          (小时，可选)
│   └── minute        (分钟，可选)
├── test_content      (测试对话内容)
├── expected_pattern  (预期响应模式，用于验证)
├── webhook_urls      (Webhook URL列表，JSON数组)
├── max_retries       (最大重试次数)
├── timeout           (超时时间，秒)
├── created_at
├── updated_at
├── last_run_at       (上次运行时间)
└── remark            (备注)

monitor_results (监控结果)
├── id
├── task_id           (关联的task ID)
├── channel_id        (测试的渠道)
├── model             (测试的模型)
├── status            (0=成功, 1=失败)
├── response_time     (响应时间，毫秒)
├── tokens_used       (消耗的token数)
├── error_message     (错误信息)
├── request_id        (用于追踪)
├── full_response     (完整响应，可选)
├── retry_count       (重试次数)
├── webhook_notified  (是否已通知webhook)
├── created_at
└── updated_at

monitor_alerts (告警配置)
├── id
├── task_id
├── alert_type        (0=连续失败次数, 1=成功率低于阈值)
├── threshold         (阈值，如连续失败3次或成功率<90%)
├── window_size       (时间窗口，分钟)
├── webhook_enabled   (是否启用webhook通知)
├── page_alert_enabled (是否启用页面提示)
├── created_at
└── updated_at
```

### 2.2 文件结构

```
new-api/
├── model/
│   └── monitor.go                    (数据模型，独立文件)
├── service/
│   ├── monitor.go                    (监控业务逻辑)
│   └── notification.go               (Webhook通知)
├── controller/
│   └── monitor.go                    (API控制器)
├── router/
│   └── monitor.go                    (路由配置)
├── web/src/
│   ├── pages/
│   │   └── Monitor/
│   │       ├── index.jsx             (主页面)
│   │       ├── TaskList.jsx          (任务列表)
│   │       ├── TaskForm.jsx          (任务编辑)
│   │       └── ResultsView.jsx       (结果展示)
│   ├── components/monitor/
│   │   ├── TaskCard.jsx
│   │   ├── ResultsChart.jsx
│   │   ├── AlertConfig.jsx
│   │   └── WebhookConfig.jsx
│   └── services/
│       └── monitor.js                (API调用)
└── migration/
    └── monitor_schema.sql            (数据库初始化)
```

## 3. API 接口设计

### 3.1 监控任务管理

#### 创建监控任务
```
POST /api/monitor/tasks
Body: {
  name: "string",
  enabled: boolean,
  channels: [1, 2, 3],
  models: ["gpt-4", "claude-3"],
  schedule: {
    interval: 300,  // 5分钟间隔
    hour: null,     // 不按小时运行
    minute: null
  },
  test_content: "请告诉我你是什么模型?",
  expected_pattern: "gpt-4|claude",  // 正则表达式，可选
  webhook_urls: ["https://example.com/webhook"],
  max_retries: 2,
  timeout: 30,
  remark: "string"
}

Response: {
  success: true,
  data: { id: 1, ... },
  message: ""
}
```

#### 获取监控任务列表
```
GET /api/monitor/tasks?page=1&limit=20

Response: {
  success: true,
  data: {
    tasks: [...],
    total: 100
  }
}
```

#### 获取监控任务详情
```
GET /api/monitor/tasks/:id

Response: {
  success: true,
  data: { ... }
}
```

#### 更新监控任务
```
PUT /api/monitor/tasks/:id
Body: { ... same as create ... }

Response: {
  success: true
}
```

#### 删除监控任务
```
DELETE /api/monitor/tasks/:id

Response: {
  success: true
}
```

#### 启用/禁用监控任务
```
PATCH /api/monitor/tasks/:id/toggle
Body: { enabled: true/false }

Response: {
  success: true
}
```

### 3.2 监控结果查询

#### 获取最新结果
```
GET /api/monitor/tasks/:id/latest-results

Response: {
  success: true,
  data: {
    results: [
      {
        channel_id: 1,
        model: "gpt-4",
        status: 0,  // 0=成功, 1=失败
        response_time: 1234,
        tokens_used: 150,
        error_message: null,
        created_at: "2024-01-01T12:00:00Z"
      }
    ]
  }
}
```

#### 获取监控结果历史
```
GET /api/monitor/tasks/:id/results?start_date=2024-01-01&end_date=2024-01-31&limit=100

Response: {
  success: true,
  data: {
    results: [...],
    total: 1000,
    statistics: {
      success_rate: 99.5,
      avg_response_time: 1200,
      total_errors: 5
    }
  }
}
```

#### 获取任务统计信息
```
GET /api/monitor/tasks/:id/statistics

Response: {
  success: true,
  data: {
    total_runs: 288,
    total_successes: 287,
    total_failures: 1,
    success_rate: 99.65,
    avg_response_time: 1250,
    min_response_time: 800,
    max_response_time: 3500,
    avg_tokens_used: 150,
    channels_status: [
      { channel_id: 1, success_rate: 100, last_run_at: "..." },
      { channel_id: 2, success_rate: 95, last_run_at: "..." }
    ]
  }
}
```

### 3.3 告警管理

#### 配置告警
```
POST /api/monitor/tasks/:id/alerts
Body: {
  alert_type: 0,      // 0=连续失败, 1=成功率低
  threshold: 3,       // 3次连续失败或<90%成功率
  window_size: 60,    // 60分钟内
  webhook_enabled: true,
  page_alert_enabled: true
}

Response: {
  success: true
}
```

#### 获取告警配置
```
GET /api/monitor/tasks/:id/alerts

Response: {
  success: true,
  data: [...]
}
```

### 3.4 Webhook 配置

#### 添加 Webhook
```
POST /api/monitor/tasks/:id/webhooks
Body: {
  url: "https://example.com/webhook",
  events: ["failure", "success"],  // 可选，默认只通知failure
  secret: "optional_signature_secret"
}

Response: {
  success: true
}
```

#### 测试 Webhook
```
POST /api/monitor/tasks/:id/webhooks/:webhook_id/test

Response: {
  success: true,
  data: {
    http_status: 200,
    response_time: 234,
    response_body: "..."
  }
}
```

#### 删除 Webhook
```
DELETE /api/monitor/tasks/:id/webhooks/:webhook_id

Response: {
  success: true
}
```

### 3.5 手动触发

#### 手动运行一次监控
```
POST /api/monitor/tasks/:id/run-now

Response: {
  success: true,
  data: {
    results: [...]  // 立即执行并返回结果
  }
}
```

## 4. 后端实现细节

### 4.1 定时任务调度
- 使用 `github.com/robfig/cron` 库
- 支持多个监控任务并发执行
- 每个任务有独立的调度器
- 支持 cron 表达式和固定间隔

### 4.2 监控执行流程
1. 从数据库读取启用的任务
2. 针对每个任务：
   - 遍历所有选中的渠道和模型组合
   - 构建真实的对话请求（不同于简单测试）
   - 调用现有的relay系统发送请求
   - 记录详细结果到数据库
3. 检测是否触发告警
4. 如果触发告警，发送Webhook通知

### 4.3 对话测试 vs 连接测试
- **连接测试**（现有）：简单验证渠道是否可访问
- **对话测试**（新增）：发送真实对话请求，检查语义响应

### 4.4 Webhook 通知格式
```json
{
  "event": "monitor_failure",
  "task_id": 1,
  "task_name": "监控任务名",
  "channel_id": 1,
  "channel_name": "OpenAI",
  "model": "gpt-4",
  "status": "failed",
  "error_message": "...",
  "response_time": 5000,
  "timestamp": "2024-01-01T12:00:00Z",
  "alert_type": "consecutive_failures",
  "failure_count": 3
}
```

## 5. 前端实现

### 5.1 页面布局
```
监控工具
├── 顶部导航/操作栏
│   ├── 创建新任务按钮
│   ├── 刷新按钮
│   └── 全局统计卡片（总任务数、运行中、异常等）
├── 主内容区
│   ├── Tab 1: 任务列表
│   │   ├── 任务卡片（可展开显示详情）
│   │   ├── 任务操作（编辑、删除、启用/禁用、手动运行）
│   │   └── 搜索/筛选
│   ├── Tab 2: 实时监控（仪表板）
│   │   ├── 运行中的任务显示
│   │   ├── 最近告警列表
│   │   └── 系统运行统计
│   └── Tab 3: 结果历史
│       ├── 详细结果表格
│       ├── 时间范围过滤
│       └── 结果导出
└── 侧边栏（可选）
    └── 快速统计信息
```

### 5.2 任务编辑表单
- 基本信息（名称、备注）
- 监控对象选择（渠道+模型多选）
- 时间表配置（固定间隔或Cron表达式）
- 测试内容编辑
- 预期响应模式（正则表达式）
- 告警配置
- Webhook配置

### 5.3 实时反馈
- 页面顶部通知栏，显示最近的告警
- 任务卡片上显示最后一次运行状态
- 支持WebSocket实时更新结果

## 6. 集成要点

### 6.1 现有系统的利用
- 使用现有的`relay`系统发送对话请求
- 使用现有的`Channel`和`Model`模型
- 利用现有的认证和日志系统
- 使用现有的数据库连接

### 6.2 数据库初始化
- 创建独立的初始化脚本
- 在程序启动时自动创建表（如果不存在）
- 支持迁移版本管理

### 6.3 后台服务启动
- 在`main.go`中初始化监控系统
- 程序启动时启动所有启用的监控任务
- 提供优雅关闭机制

## 7. 独立性设计

### 不修改现有代码
- 所有新代码都在独立文件中
- 通过导入新的路由模块在main.go中注册
- 通过Service层调用，不直接修改controller
- 数据模型完全独立

### 易于合并
- 如果上游更新main.go，只需在新版本中注册路由
- 新增的表完全独立，不影响现有数据
- 可以通过环境变量启用/禁用监控功能

## 8. 部署配置

### 环境变量
```
MONITOR_ENABLED=true                    # 启用监控功能
MONITOR_DB_AUTO_INIT=true              # 自动初始化数据库
MONITOR_WEBHOOK_TIMEOUT=10             # Webhook超时时间
MONITOR_WEBHOOK_RETRY=3                # Webhook重试次数
MONITOR_MAX_CONCURRENT_TASKS=5         # 最大并发任务数
```

### 服务依赖
- 使用现有的数据库连接
- 使用现有的日志系统
- 依赖现有的relay系统

