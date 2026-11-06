package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Mode string `yaml:"mode"` // debug / release
	} `yaml:"server"`

	Database struct {
		Driver string `yaml:"driver"` // mysql / sqlite
		DSN    string `yaml:"dsn"`
	} `yaml:"database"`

	JWT struct {
		Secret      string `yaml:"secret"`
		ExpireHours int    `yaml:"expire_hours"`
	} `yaml:"jwt"`

	Push struct {
		RequirePhone     bool `yaml:"require_phone"`      // 第三方推送 phone 是否必填
		MaxBodySizeBytes int64 `yaml:"max_body_size_bytes"` // 单次请求体上限
	} `yaml:"push"`

	Mail struct {
		WorkerCount       int `yaml:"worker_count"`        // 并发 worker 数
		PollIntervalSec   int `yaml:"poll_interval_sec"`   // 任务轮询间隔（秒）
		RetryMax          int `yaml:"retry_max"`           // 失败后额外重试次数（默认 3）
		RetryBackoffSec   int `yaml:"retry_backoff_sec"`   // 重试基础退避（秒），按次数指数增长
		SendTimeoutSec    int `yaml:"send_timeout_sec"`    // 单封 SMTP 发送超时
	} `yaml:"mail"`

	App struct {
		BaseURL string `yaml:"base_url"` // 对外展示的接口地址前缀，如 http://127.0.0.1:8080
	} `yaml:"app"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path == "" {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	applyDefaults(cfg)
	return cfg, nil
}

func applyDefaults(c *Config) {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "debug"
	}
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.JWT.ExpireHours <= 0 {
		c.JWT.ExpireHours = 24
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "web-mailer-default-secret-change-me"
	}
	if c.Push.MaxBodySizeBytes <= 0 {
		c.Push.MaxBodySizeBytes = 1 << 20 // 1MB
	}
	if c.Mail.WorkerCount <= 0 {
		c.Mail.WorkerCount = 3
	}
	if c.Mail.PollIntervalSec <= 0 {
		c.Mail.PollIntervalSec = 5
	}
	if c.Mail.RetryMax <= 0 {
		c.Mail.RetryMax = 3
	}
	if c.Mail.RetryBackoffSec <= 0 {
		c.Mail.RetryBackoffSec = 60
	}
	if c.Mail.SendTimeoutSec <= 0 {
		c.Mail.SendTimeoutSec = 30
	}
	if c.App.BaseURL == "" {
		c.App.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", c.Server.Port)
	}
}
