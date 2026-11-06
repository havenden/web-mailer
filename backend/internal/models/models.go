package models

import "time"

// ---------- 用户 ----------

type SysUser struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"size:100;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255" json:"-"`
	Nickname     string     `gorm:"size:100" json:"nickname"`
	Email        string     `gorm:"size:255;index" json:"email"`
	Phone        string     `gorm:"size:30" json:"phone"`
	Role         string     `gorm:"size:30;default:admin" json:"role"` // admin / user
	Status       int8       `gorm:"default:1" json:"status"`           // 1 启用 0 禁用
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"-"`
}

func (SysUser) TableName() string { return "sys_users" }

// ---------- 接口配置 ----------

type SysInterface struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                 string     `gorm:"size:100" json:"name"`
	InterfaceKey         string     `gorm:"size:64;uniqueIndex;not null" json:"interface_key"`
	EndpointURL          string     `gorm:"size:255" json:"endpoint_url"`
	Region               string     `gorm:"size:100" json:"region"`
	QualificationAddress string     `gorm:"size:255" json:"qualification_address"`
	Channel              string     `gorm:"size:100" json:"channel"`
	Status               int8       `gorm:"default:1" json:"status"`
	Remark               string     `gorm:"type:text" json:"remark"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `gorm:"index" json:"-"`
}

func (SysInterface) TableName() string { return "sys_interfaces" }

// ---------- 接口-用户 多对多 ----------

type SysInterfaceUser struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	InterfaceID int64     `gorm:"index;not null" json:"interface_id"`
	UserID      int64     `gorm:"index;not null" json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SysInterfaceUser) TableName() string { return "sys_interface_users" }

// ---------- QQ 邮箱 SMTP 配置 ----------

type SysMailConfig struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigName   string    `gorm:"size:100" json:"config_name"`
	SMTPHost     string    `gorm:"size:100;default:smtp.qq.com" json:"smtp_host"`
	SMTPPort     int       `gorm:"default:465" json:"smtp_port"`
	SenderEmail  string    `gorm:"size:255" json:"sender_email"`
	SenderName   string    `gorm:"size:100" json:"sender_name"`
	SMTPUsername string    `gorm:"size:255" json:"smtp_username"`
	AuthCodeEnc  string    `gorm:"size:512" json:"-"` // 授权码加密存储，不向前端输出
	Encryption   string    `gorm:"size:20;default:ssl" json:"encryption"` // ssl / starttls
	IsDefault    int8      `gorm:"default:0" json:"is_default"`
	Status       int8      `gorm:"default:1" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (SysMailConfig) TableName() string { return "sys_mail_configs" }

// HasAuthCode 便于前端显示「已配置授权码」
func (c *SysMailConfig) HasAuthCode() bool { return c.AuthCodeEnc != "" }

// ---------- 邮件模板 ----------

type SysMailTemplate struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"size:100" json:"name"`
	SubjectTemplate  string    `gorm:"size:255" json:"subject_template"`
	HTMLTemplate     string    `gorm:"type:longtext" json:"html_template"`
	IsDefault        int8      `gorm:"default:0" json:"is_default"`
	Status           int8      `gorm:"default:1" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (SysMailTemplate) TableName() string { return "sys_mail_templates" }

// ---------- 数据接收记录 ----------

type SysPushRecord struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	InterfaceID  int64     `gorm:"index" json:"interface_id"`
	Sequence     string    `gorm:"size:32;index" json:"sequence"` // yyyyMMddHHmmss
	RawJSON      string    `gorm:"type:longtext" json:"raw_json"`
	Phone        string    `gorm:"size:30;index" json:"phone"`
	Name         string    `gorm:"size:100" json:"name"`
	Channel      string    `gorm:"size:100" json:"channel"`
	ProcessNote  string    `gorm:"size:255" json:"process_note"` // 如「无可用收件用户」
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (SysPushRecord) TableName() string { return "sys_push_records" }

// ---------- 邮件任务/记录 ----------

type SysMailRecord struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	PushRecordID   int64      `gorm:"index;not null" json:"push_record_id"`
	InterfaceID    int64      `gorm:"index" json:"interface_id"`
	UserID         int64      `gorm:"index" json:"user_id"`
	RecipientEmail string     `gorm:"size:255;index" json:"recipient_email"`
	RecipientName  string     `gorm:"size:100" json:"recipient_name"` // 发送时昵称快照
	Subject        string     `gorm:"size:255" json:"subject"`
	HTMLContent    string     `gorm:"type:longtext" json:"html_content"`
	Status         int8       `gorm:"index;default:0" json:"status"` // 0待发送 1发送中 2成功 3重试等待 4失败
	AttemptCount   int        `gorm:"default:0" json:"attempt_count"`
	LastError      string     `gorm:"type:text" json:"last_error"`
	LastSentAt     *time.Time `json:"last_sent_at"`
	NextRetryAt    *time.Time `gorm:"index" json:"next_retry_at"`
	SentAt         *time.Time `json:"sent_at"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (SysMailRecord) TableName() string { return "sys_mail_records" }

// 状态常量
const (
	MailStatusPending  int8 = 0 // 待发送
	MailStatusSending  int8 = 1 // 发送中
	MailStatusSuccess  int8 = 2 // 发送成功
	MailStatusRetry    int8 = 3 // 重试等待
	MailStatusFailed   int8 = 4 // 发送失败（超过重试上限）
)

// ---------- 操作日志 ----------

type SysOperationLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OperatorID  int64     `json:"operator_id"`
	Operator    string    `gorm:"size:100" json:"operator"`
	Module      string    `gorm:"size:50" json:"module"`
	Action      string    `gorm:"size:50" json:"action"`
	Description string    `gorm:"size:500" json:"description"`
	Result      string    `gorm:"size:20" json:"result"` // success / fail
	SourceIP    string    `gorm:"size:50" json:"source_ip"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (SysOperationLog) TableName() string { return "sys_operation_logs" }

// ---------- 登录日志 ----------

type SysLoginLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `json:"user_id"`
	Username    string    `gorm:"size:100" json:"username"`
	LoginResult string    `gorm:"size:20" json:"login_result"` // success / fail
	Message     string    `gorm:"size:255" json:"message"`
	SourceIP    string    `gorm:"size:50" json:"source_ip"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (SysLoginLog) TableName() string { return "sys_login_logs" }
