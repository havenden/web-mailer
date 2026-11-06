package services

import (
	"bytes"
	"encoding/json"
	"html"
	"log"
	"strings"
	"time"

	"web-mailer/internal/config"
	"web-mailer/internal/models"
)

// Worker 基于 MySQL 持久化任务表 + 轮询领取，服务重启后可继续处理待发送任务。
type Worker struct {
	cfg   *config.Config
	stop  chan struct{}
}

func NewWorker(cfg *config.Config) *Worker {
	return &Worker{cfg: cfg, stop: make(chan struct{})}
}

func (w *Worker) Start() {
	for i := 0; i < w.cfg.Mail.WorkerCount; i++ {
		go w.loop(i)
	}
	log.Printf("[Worker] 已启动 %d 个邮件发送 Worker，轮询间隔 %ds", w.cfg.Mail.WorkerCount, w.cfg.Mail.PollIntervalSec)
}

func (w *Worker) Stop() { close(w.stop) }

func (w *Worker) loop(id int) {
	interval := time.Duration(w.cfg.Mail.PollIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.processOne(id)
		}
	}
}

// processOne 领取一条待发送任务并处理
func (w *Worker) processOne(workerID int) {
	now := time.Now()
	var rec models.SysMailRecord
	err := models.DB.Where(
		"status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		[]int8{models.MailStatusPending, models.MailStatusRetry}, now,
	).Order("id asc").First(&rec).Error
	if err != nil {
		return // 无任务
	}

	// 乐观领取：原子更新状态，避免多 Worker 领取同一条
	res := models.DB.Model(&models.SysMailRecord{}).
		Where("id = ? AND status IN ?", rec.ID, []int8{models.MailStatusPending, models.MailStatusRetry}).
		Updates(map[string]interface{}{"status": models.MailStatusSending, "updated_at": now})
	if res.RowsAffected == 0 {
		return // 已被其他 Worker 领取
	}

	w.send(&rec)
}

func (w *Worker) send(rec *models.SysMailRecord) {
	attempt := rec.AttemptCount + 1
	rec.AttemptCount = attempt
	now := time.Now()
	rec.LastSentAt = &now

	// 发送时快照：按当前用户昵称重新渲染（避免用户资料修改后与历史记录不一致的场景由快照字段保证）
	user := models.SysUser{}
	nickname := rec.RecipientName
	if err := models.DB.First(&user, rec.UserID).Error; err == nil && user.Nickname != "" {
		nickname = user.Nickname
	}

	// 获取接口配置与原始数据
	push := models.SysPushRecord{}
	if err := models.DB.First(&push, rec.PushRecordID).Error; err != nil {
		w.fail(rec, attempt, "推送记录不存在: "+err.Error())
		return
	}
	iface := models.SysInterface{}
	if err := models.DB.Unscoped().First(&iface, rec.InterfaceID).Error; err != nil {
		w.fail(rec, attempt, "接口配置不存在: "+err.Error())
		return
	}

	data := BuildMailData(&iface, &push, nickname)

	// 默认模板
	tpl := models.SysMailTemplate{}
	if err := models.DB.Where("is_default = 1 AND status = 1").First(&tpl).Error; err != nil {
		w.fail(rec, attempt, "未找到可用的默认邮件模板: "+err.Error())
		return
	}

	subject, err := RenderSubject(tpl.SubjectTemplate, data)
	if err != nil {
		w.fail(rec, attempt, "渲染邮件标题失败: "+err.Error())
		return
	}
	htmlBody, err := RenderHTML(tpl.HTMLTemplate, data)
	if err != nil {
		w.fail(rec, attempt, "渲染邮件正文失败: "+err.Error())
		return
	}

	// 保存渲染快照
	rec.Subject = subject
	rec.HTMLContent = htmlBody
	rec.RecipientName = nickname

	// 默认发件配置
	mc := models.SysMailConfig{}
	if err := models.DB.Where("is_default = 1 AND status = 1").First(&mc).Error; err != nil {
		w.fail(rec, attempt, "未找到可用的默认邮件配置（请先在邮件配置中设置默认配置并启用）: "+err.Error())
		return
	}
	authCode, err := DecryptAuthCode(mc.AuthCodeEnc, w.cfg.JWT.Secret)
	if err != nil {
		w.fail(rec, attempt, "读取 SMTP 授权码失败: "+err.Error())
		return
	}

	if err := SendMail(&mc, authCode, rec.RecipientEmail, subject, htmlBody); err != nil {
		// 发送失败 → 判断是否还需重试（首次 + 额外 retry_max 次）
		maxAttempts := 1 + w.cfg.Mail.RetryMax
		if attempt >= maxAttempts {
			w.fail(rec, attempt, err.Error())
			return
		}
		backoff := time.Duration(w.cfg.Mail.RetryBackoffSec*attempt) * time.Second
		next := time.Now().Add(backoff)
		models.DB.Model(rec).Updates(map[string]interface{}{
			"status":        models.MailStatusRetry,
			"attempt_count": attempt,
			"last_error":    truncateErr(err.Error()),
			"last_sent_at":  now,
			"next_retry_at": next,
		})
		log.Printf("[Worker] 邮件 #%d 发送失败（第 %d/%d 次）：%s；%.0f 秒后重试", rec.ID, attempt, maxAttempts, err, backoff.Seconds())
		return
	}

	sentAt := time.Now()
	models.DB.Model(rec).Updates(map[string]interface{}{
		"status":        models.MailStatusSuccess,
		"attempt_count": attempt,
		"last_error":    "",
		"subject":       rec.Subject,
		"html_content":  rec.HTMLContent,
		"recipient_name": rec.RecipientName,
		"last_sent_at":  now,
		"sent_at":       sentAt,
		"next_retry_at": nil,
	})
	log.Printf("[Worker] 邮件 #%d 已发送至 %s（第 %d 次尝试成功）", rec.ID, rec.RecipientEmail, attempt)
}

func (w *Worker) fail(rec *models.SysMailRecord, attempt int, reason string) {
	models.DB.Model(rec).Updates(map[string]interface{}{
		"status":        models.MailStatusFailed,
		"attempt_count": attempt,
		"last_error":    truncateErr(reason),
		"next_retry_at": nil,
	})
	log.Printf("[Worker] 邮件 #%d 最终发送失败：%s", rec.ID, reason)
}

func truncateErr(s string) string {
	r := []rune(s)
	if len(r) > 1000 {
		return string(r[:1000])
	}
	return s
}

// BuildMailData 组装邮件渲染数据
// 序号缺失时按接收时间兜底；号码/病情/医生优先取推送原始 JSON（兼容各渠道不同字段命名）
func BuildMailData(iface *models.SysInterface, push *models.SysPushRecord, nickname string) *MailData {
	extra := ExtractPushExtra(push.RawJSON)

	sequence := strings.TrimSpace(push.Sequence)
	if sequence == "" {
		sequence = push.CreatedAt.Format("20060102150405")
	}
	channel := push.Channel
	if channel == "" {
		channel = iface.Channel
	}
	phone := strings.TrimSpace(push.Phone)
	if phone == "" {
		phone = extra["phone"]
	}
	name := nickname
	if name == "" {
		name = extra["name"]
	}

	return &MailData{
		Region:               iface.Region,
		Sequence:             sequence,
		Channel:              channel,
		Phone:                phone,
		Condition:            extra["condition"],
		Nickname:             name,
		Doctor:               extra["doctor"],
		QualificationAddress: iface.QualificationAddress,
		InterfaceName:        iface.Name,
		RawJSONHTML:          RawJSONHTML(push.RawJSON),
	}
}

// RawJSONHTML 把原始 JSON 美化、转义并转为正文可用的 HTML（换行 → <br>）
func RawJSONHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	pretty := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err == nil {
		pretty = buf.String()
	}
	esc := html.EscapeString(pretty)
	return strings.ReplaceAll(esc, "\n", "<br>")
}

// MailCopyText 生成某条邮件记录的「序号 → 资质所属地」复制文本（与邮件正文一致）
func MailCopyText(rec *models.SysMailRecord) (string, error) {
	var push models.SysPushRecord
	if err := models.DB.First(&push, rec.PushRecordID).Error; err != nil {
		return "", err
	}
	var iface models.SysInterface
	if err := models.DB.Unscoped().First(&iface, rec.InterfaceID).Error; err != nil {
		return "", err
	}
	nickname := rec.RecipientName
	var u models.SysUser
	if err := models.DB.First(&u, rec.UserID).Error; err == nil && u.Nickname != "" {
		nickname = u.Nickname
	}
	return BuildCopyText(BuildMailData(&iface, &push, nickname)), nil
}

// RetryMail 手动重发：把最终失败的任务重置为待发送（重置尝试计数）
func RetryMail(recordID int64) error {
	res := models.DB.Model(&models.SysMailRecord{}).
		Where("id = ? AND status IN ?", recordID, []int8{models.MailStatusFailed, models.MailStatusRetry}).
		Updates(map[string]interface{}{
			"status":        models.MailStatusPending,
			"attempt_count": 0,
			"last_error":    "",
			"next_retry_at": nil,
		})
	if res.RowsAffected == 0 {
		return errRecordNotRetryable
	}
	return nil
}
