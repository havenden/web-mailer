# Web-Mailer

接口数据接收与邮件推送管理系统。第三方通过 HTTP 推送线索数据，系统自动渲染邮件模板并通过 SMTP 发送给绑定用户。

## 技术栈

- **后端**：Go + Gin + GORM（支持 MySQL 8 / SQLite）
- **前端**：Vue 3 + Vite + Ant Design Vue 4
- **邮件**：QQ 邮箱 SMTP（SSL 465 / STARTTLS 587），独立发送，失败自动重试

## 功能

- 仪表盘：数据统计卡片、7 天趋势图、接口排行
- 用户管理：CRUD、角色分配、启用/禁用
- 接口管理：生成唯一推送地址、绑定用户（多对多）
- 邮件配置：SMTP 配置管理、授权码 AES-GCM 加密存储、测试发送
- 邮件模板：HTML 模板编辑、变量渲染、实时预览
- 接收记录：推送数据查看、关联邮件任务
- 邮件记录：发送状态追踪、手动重发、一键复制正文
- 操作日志：全操作审计

## 快速启动

```bash
# 后端（SQLite 模式，首次启动自动建库并创建管理员）
cd backend
go run .
# 服务运行在 http://127.0.0.1:8806

# 前端（开发模式）
cd frontend
npm install
npm run dev
# 访问 http://localhost:5173
```

默认管理员：`admin` / `admin123`（登录后请立即修改密码）

## 生产部署

1. 修改 `backend/config.yaml`：

```yaml
database:
  driver: mysql
  dsn: "root:yourpassword@tcp(127.0.0.1:3306)/webmailer?charset=utf8mb4&parseTime=True&loc=Local"
jwt:
  secret: "换成长随机字符串"
app:
  base_url: "https://你的域名"
```

2. 构建：

```bash
cd frontend && npm run build
cd ../backend
CGO_ENABLED=0 go build -o web-mailer
./web-mailer
```

单端口部署：静态页面 + API 同在 8806。

## 第三方推送

```bash
curl -X POST http://127.0.0.1:8806/api/v1/push/{interface_key} \
  -H "Content-Type: application/json" \
  -d '{"name":"张三","phone":"13800000000","cityLocation":"重庆-重庆","clueSource":"本地推"}'
```

响应示例：

```json
{"code":200,"message":"数据接收成功","data":{"record_id":1,"sequence":"20261106150035","mail_tasks":1}}
```

推送接口无认证、无频率限制，仅通过 `interface_key` 路由。

## 核心规则

- 每次推送均为新数据，不去重
- 接口与用户多对多；每条数据为每个绑定用户创建独立邮件任务
- 邮件标题：`【线索推送】{地域} - {接口名称}`
- 授权码 AES-GCM 加密存储，前端永不回显
- Worker 失败后额外重试 3 次（退避递增），支持手动重发
- 推送字段宽松解析，兼容多种命名（phone/mobile/tel 等）

## 配置项

| 配置 | 默认 | 说明 |
|---|---|---|
| mail.worker_count | 3 | 发送 Worker 并发数 |
| mail.retry_max | 3 | 首次失败后额外重试次数 |
| mail.retry_backoff_sec | 60 | 重试退避基数（秒） |
| push.require_phone | true | phone 字段是否必填 |
| push.max_body_size_bytes | 1MB | 单次请求体上限 |
