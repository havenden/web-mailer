package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"web-mailer/internal/config"
	"web-mailer/internal/models"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	var err error
	switch cfg.Database.Driver {
	case "mysql":
		DB, err = gorm.Open(mysql.Open(cfg.Database.DSN), gormCfg)
	case "sqlite", "":
		dbPath := cfg.Database.DSN
		if dbPath == "" {
			dbPath = "webmailer.db"
		}
		DB, err = gorm.Open(sqlite.Open(dbPath), gormCfg)
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", cfg.Database.Driver)
	}
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	models.DB = DB

	if err := DB.AutoMigrate(
		&models.SysUser{},
		&models.SysInterface{},
		&models.SysInterfaceUser{},
		&models.SysMailConfig{},
		&models.SysMailTemplate{},
		&models.SysPushRecord{},
		&models.SysMailRecord{},
		&models.SysOperationLog{},
		&models.SysLoginLog{},
	); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	if err := seed(cfg); err != nil {
		return fmt.Errorf("初始化数据失败: %w", err)
	}
	return nil
}

// seed 初始化默认管理员与默认邮件模板
func seed(cfg *config.Config) error {
	var userCount int64
	DB.Model(&models.SysUser{}).Count(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.SysUser{
			Username:     "admin",
			PasswordHash: string(hash),
			Nickname:     "管理员",
			Email:        "admin@example.com",
			Role:         "admin",
			Status:       1,
		}
		if err := DB.Create(&admin).Error; err != nil {
			return err
		}
		log.Printf("已创建默认管理员 admin / admin123（请登录后修改密码）")
	}

	var tplCount int64
	DB.Model(&models.SysMailTemplate{}).Count(&tplCount)
	if tplCount == 0 {
		tpl := models.SysMailTemplate{
			Name:            "默认模板",
			SubjectTemplate: "【线索推送】{{.Region}} - {{.InterfaceName}}",
			HTMLTemplate:    DefaultHTMLTemplate,
			IsDefault:       1,
			Status:          1,
		}
		if err := DB.Create(&tpl).Error; err != nil {
			return err
		}
	} else {
		// 旧版本默认模板自动升级（仅升级系统生成的模板，不影响用户自定义模板）
		var tpls []models.SysMailTemplate
		DB.Find(&tpls)
		for i := range tpls {
			h := tpls[i].HTMLTemplate
			needUpgrade := (strings.Contains(h, legacyTemplateMarker) || h == defaultHTMLTemplateV2) &&
				!strings.Contains(h, "{{.RawJSONHTML}}")
			if needUpgrade {
				if err := DB.Model(&tpls[i]).Update("html_template", DefaultHTMLTemplate).Error; err != nil {
					return err
				}
				log.Printf("已升级邮件模板 #%d「%s」：序号→资质所属地 整段复制 + 底部完整 JSON", tpls[i].ID, tpls[i].Name)
			}
		}
	}

	// 文案调整：数据推送 → 线索推送（同时修正已存在的模板标题与正文）
	if err := DB.Model(&models.SysMailTemplate{}).
		Where("subject_template LIKE ? OR html_template LIKE ?", "%数据推送%", "%数据推送%").
		Updates(map[string]interface{}{
			"subject_template": gorm.Expr("REPLACE(subject_template, '数据推送', '线索推送')"),
			"html_template":    gorm.Expr("REPLACE(html_template, '数据推送', '线索推送')"),
		}).Error; err != nil {
		return err
	}

	// 标题改用接口名称：{{.Channel}} → {{.InterfaceName}}（仅标题模板）
	if err := DB.Model(&models.SysMailTemplate{}).
		Where("subject_template LIKE ?", "%{{.Channel}}%").
		Update("subject_template", gorm.Expr("REPLACE(subject_template, '{{.Channel}}', '{{.InterfaceName}}')")).Error; err != nil {
		return err
	}

	// SQLite 仅用于本地演示
	if cfg.Database.Driver != "mysql" {
		log.Printf("[提示] 当前使用 SQLite 本地模式（%s），生产环境请在 config.yaml 中配置 MySQL", cfg.Database.DSN)
		_ = os.Stdout
	}
	return nil
}

// legacyTemplateMarker 旧版默认模板特征（用于启动时自动升级）
const legacyTemplateMarker = "border-radius:8px;line-height:2;"

// defaultHTMLTemplateV2 上一版默认模板（精确匹配，用于自动升级到最新版）
const defaultHTMLTemplateV2 = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>数据推送通知</title>
</head>
<body style="margin:0;padding:20px;background:#f5f6f8;">
  <div style="max-width:600px;margin:0 auto;padding:24px;font-family:Arial,'Microsoft YaHei',sans-serif;color:#333;background:#fff;">
    <h2 style="font-size:20px;margin:0 0 14px;">数据推送通知</h2>
    <div style="padding:12px 16px;background:#f7f8fa;border-radius:8px;line-height:26px;font-size:14px;">
      <strong>【地域】</strong>{{.Region}}
    </div>
    <div style="margin-top:12px;padding:12px 16px;background:#f7f8fa;border-radius:8px;line-height:26px;font-size:14px;"><strong>【序号】</strong>{{.Sequence}}<br><strong>【渠道】</strong>{{.Channel}}<br><strong>【号码】</strong>{{.Phone}}<br><strong>【病情】</strong>{{.Condition}}<br><strong>【运营】</strong>{{.Nickname}}<br><strong>【医生】</strong>{{.Doctor}}<br><strong>【资质所属地】</strong>{{.QualificationAddress}}</div>
    <p style="margin:12px 0 0;color:#999;font-size:12px;">提示：以上灰底内容可整段选中、一次性复制。</p>
    <p style="margin:14px 0 0;color:#999;font-size:12px;">本邮件由 Web-Mailer 系统自动发送，请勿直接回复。</p>
  </div>
</body>
</html>`

// DefaultHTMLTemplate 默认邮件正文模板
// 上半部分：序号 → 资质所属地 用 <br> 逐行排列，整段选中复制即得干净的 7 行纯文本；
// 最下面：附「完整数据」深色块，展示第三方推送的原始 JSON（美化 + 转义）。
const DefaultHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>线索推送通知</title>
</head>
<body style="margin:0;padding:20px;background:#f5f6f8;">
  <div style="max-width:600px;margin:0 auto;padding:24px;font-family:Arial,'Microsoft YaHei',sans-serif;color:#333;background:#fff;">
    <h2 style="font-size:20px;margin:0 0 14px;">线索推送通知</h2>
    <div style="padding:12px 16px;background:#f7f8fa;border-radius:8px;line-height:26px;font-size:14px;">
      <strong>【地域】</strong>{{.Region}}
    </div>
    <div style="margin-top:12px;padding:12px 16px;background:#f7f8fa;border-radius:8px;line-height:26px;font-size:14px;"><strong>【序号】</strong>{{.Sequence}}<br><strong>【渠道】</strong>{{.Channel}}<br><strong>【号码】</strong>{{.Phone}}<br><strong>【病情】</strong>{{.Condition}}<br><strong>【运营】</strong>{{.Nickname}}<br><strong>【医生】</strong>{{.Doctor}}<br><strong>【资质所属地】</strong>{{.QualificationAddress}}</div>
    <p style="margin:12px 0 0;color:#999;font-size:12px;">提示：以上灰底内容可整段选中、一次性复制。</p>
    <p style="margin:16px 0 6px;font-size:13px;color:#303133;font-weight:bold;">完整数据（原始 JSON）</p>
    <div style="padding:12px 14px;background:#1e2430;color:#d6e2f0;border-radius:8px;font-family:Consolas,Menlo,monospace;font-size:12px;line-height:1.9;white-space:pre-wrap;word-break:break-all;">{{.RawJSONHTML}}</div>
    <p style="margin:14px 0 0;color:#999;font-size:12px;">本邮件由 Web-Mailer 系统自动发送，请勿直接回复。</p>
  </div>
</body>
</html>`
