# Alertmanager → Gotify 格式转换中间件

轻量 HTTP 服务，用于将 Alertmanager 的 Webhook 告警格式，转换为 Gotify 可识别的消息格式，实现告警实时推送。

## 核心功能
- 自动解析 Alertmanager 告警字段（状态、级别、描述、时间）
- 按 severity 映射 Gotify 消息优先级（critical→8、warning→5、info→3）
- 时间自动转为本地时区，更易读
- 支持告警触发（firing）和恢复（resolved）状态推送

## 快速部署

### 1. 环境依赖
- Go 1.11+（用于编译）
- Gotify 服务器（已部署并获取应用令牌）

### 2. 编译代码
```bash
# 进入项目目录
cd /opt/test/alert2gotify
# 初始化 Go 模块（已执行过可跳过）
go mod init alert2gotify
# 编译生成可执行文件
go build -o alert2gotify .
```

### 3. 运行中间件
```bash
# 替换为你的 Gotify 地址和应用令牌以及监听端口
GOTIFY_URL="http://10.10.11.1:9406" GOTIFY_TOKEN="AkBKingi2FPq86o" LISTEN_PORT="9407" ./alert2gotify
```

### 4. 配置alertmanager.yml
```bash
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'gotify'
receivers:
- name: 'gotify'
  webhook_configs:
  - url: 'http://10.10.11.1:9407/webhook'  # 替换为中间件实际地址
    send_resolved: true  # 推送告警恢复状态
    http_config:
      timeout: 10s
```

### 5.验证
```bash
curl -X POST http://localhost:9407/webhook   -H "Content-Type: application/json"   -d @test.json
```