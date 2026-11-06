package handlers

import (
	"github.com/gin-gonic/gin"

	"web-mailer/internal/models"
	"web-mailer/internal/services"
)

// GET /api/v1/mail/configs
func ListMailConfigs(c *gin.Context) {
	var list []models.SysMailConfig
	models.DB.Order("id desc").Find(&list)
	items := make([]gin.H, 0, len(list))
	for _, m := range list {
		items = append(items, gin.H{
			"id": m.ID, "config_name": m.ConfigName, "smtp_host": m.SMTPHost,
			"smtp_port": m.SMTPPort, "sender_email": m.SenderEmail, "sender_name": m.SenderName,
			"smtp_username": m.SMTPUsername, "encryption": m.Encryption,
			"is_default": m.IsDefault, "status": m.Status,
			"has_auth_code": m.HasAuthCode(),
			"created_at": m.CreatedAt, "updated_at": m.UpdatedAt,
		})
	}
	ok(c, gin.H{"list": items})
}

type mailConfigPayload struct {
	ConfigName   string `json:"config_name"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SenderEmail  string `json:"sender_email"`
	SenderName   string `json:"sender_name"`
	SMTPUsername string `json:"smtp_username"`
	AuthCode     string `json:"auth_code"`
	Encryption   string `json:"encryption"`
	IsDefault    *int8  `json:"is_default"`
	Status       *int8  `json:"status"`
}

// POST /api/v1/mail/configs
func CreateMailConfig(c *gin.Context) {
	var p mailConfigPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.ConfigName == "" || p.SenderEmail == "" || p.SMTPUsername == "" || p.AuthCode == "" {
		fail(c, 400, "配置名称、发件邮箱、SMTP 账号、授权码均不能为空")
		return
	}
	if p.SMTPHost == "" {
		p.SMTPHost = "smtp.qq.com"
	}
	if p.SMTPPort == 0 {
		p.SMTPPort = 465
	}
	if p.Encryption == "" {
		p.Encryption = "ssl"
	}
	enc, err := services.EncryptAuthCode(p.AuthCode, getAppConfig(c).JWT.Secret)
	if err != nil {
		fail(c, 500, "授权码加密保存失败")
		return
	}
	m := models.SysMailConfig{
		ConfigName: p.ConfigName, SMTPHost: p.SMTPHost, SMTPPort: p.SMTPPort,
		SenderEmail: p.SenderEmail, SenderName: p.SenderName, SMTPUsername: p.SMTPUsername,
		AuthCodeEnc: enc, Encryption: p.Encryption, Status: 1,
	}
	if p.IsDefault != nil {
		m.IsDefault = *p.IsDefault
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if err := models.DB.Create(&m).Error; err != nil {
		fail(c, 500, "创建邮件配置失败")
		return
	}
	setDefaultIfRequested(c, m.ID, m.IsDefault)
	okMsg(c, "邮件配置创建成功", gin.H{"id": m.ID})
}

// PUT /api/v1/mail/configs/:id
func UpdateMailConfig(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var m models.SysMailConfig
	if err := models.DB.First(&m, id).Error; err != nil {
		fail(c, 404, "邮件配置不存在")
		return
	}
	var p mailConfigPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.ConfigName != "" {
		m.ConfigName = p.ConfigName
	}
	if p.SMTPHost != "" {
		m.SMTPHost = p.SMTPHost
	}
	if p.SMTPPort > 0 {
		m.SMTPPort = p.SMTPPort
	}
	if p.SenderEmail != "" {
		m.SenderEmail = p.SenderEmail
	}
	m.SenderName = p.SenderName
	if p.SMTPUsername != "" {
		m.SMTPUsername = p.SMTPUsername
	}
	if p.Encryption != "" {
		m.Encryption = p.Encryption
	}
	// 授权码：未填写则保留原授权码
	if p.AuthCode != "" {
		enc, err := services.EncryptAuthCode(p.AuthCode, getAppConfig(c).JWT.Secret)
		if err != nil {
			fail(c, 500, "授权码加密保存失败")
			return
		}
		m.AuthCodeEnc = enc
	}
	if p.IsDefault != nil {
		m.IsDefault = *p.IsDefault
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	if err := models.DB.Save(&m).Error; err != nil {
		fail(c, 500, "保存邮件配置失败")
		return
	}
	setDefaultIfRequested(c, m.ID, m.IsDefault)
	okMsg(c, "邮件配置更新成功", nil)
}

func setDefaultIfRequested(c *gin.Context, id int64, isDefault int8) {
	if isDefault == 1 {
		models.DB.Model(&models.SysMailConfig{}).Where("id <> ?", id).Update("is_default", 0)
	}
}

// DELETE /api/v1/mail/configs/:id
func DeleteMailConfig(c *gin.Context) {
	id := atoi64(c.Param("id"))
	if err := models.DB.Delete(&models.SysMailConfig{}, id).Error; err != nil {
		fail(c, 500, "删除邮件配置失败")
		return
	}
	okMsg(c, "邮件配置已删除", nil)
}

// POST /api/v1/mail/test 发送测试邮件
func SendTestMail(c *gin.Context) {
	var p struct {
		ConfigID int64  `json:"config_id" binding:"required"`
		To       string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "请填写配置与有效的测试收件地址")
		return
	}
	var m models.SysMailConfig
	if err := models.DB.First(&m, p.ConfigID).Error; err != nil {
		fail(c, 404, "邮件配置不存在")
		return
	}
	authCode, err := services.DecryptAuthCode(m.AuthCodeEnc, getAppConfig(c).JWT.Secret)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	html := "<div style=\"padding:20px;font-family:Arial,'Microsoft YaHei',sans-serif;\">" +
		"<h2>Web-Mailer 测试邮件</h2><p>这是来自 Web-Mailer 的测试邮件，收到即表示 SMTP 配置有效。</p>" +
		"<p style=\"color:#999\">发件配置：" + m.ConfigName + "（" + m.SenderEmail + "）</p></div>"
	subject := "【Web-Mailer】测试邮件"
	if err := services.SendMail(&m, authCode, p.To, subject, html); err != nil {
		fail(c, 500, "测试邮件发送失败: "+err.Error())
		return
	}
	okMsg(c, "测试邮件已发送至 "+p.To, nil)
}

// POST /api/v1/mail/preview 用样例数据预览模板渲染效果（返回 HTML 与可复制文本）
func PreviewTemplate(c *gin.Context) {
	var p struct {
		SubjectTemplate string `json:"subject_template"`
		HTMLTemplate    string `json:"html_template"`
	}
	if err := c.ShouldBindJSON(&p); err != nil || p.HTMLTemplate == "" {
		fail(c, 400, "请先填写 HTML 模板")
		return
	}
	data := &services.MailData{
		Region:               "重庆",
		Sequence:             "20261106153600",
		Channel:              "抖音本地推",
		Phone:                "13800001111",
		Condition:            "内科",
		Nickname:             "陈金林",
		Doctor:               "李医生",
		QualificationAddress: "黑龙江省-佳木斯市",
		InterfaceName:        "重庆-内科线索",
		RawJSONHTML: services.RawJSONHTML(`{"advertiserName":"MK-某某医院-96","cityLocation":"重庆-重庆","clueCreatetime":1793947228000,"name":"未命名","phone":"13800001111","phoneLocation":"重庆-重庆","planName":"自动投放_10_29"}`),
	}
	subject, err := services.RenderSubject(p.SubjectTemplate, data)
	if err != nil {
		fail(c, 400, "标题模板解析失败: "+err.Error())
		return
	}
	body, err := services.RenderHTML(p.HTMLTemplate, data)
	if err != nil {
		fail(c, 400, "正文模板解析失败: "+err.Error())
		return
	}
	ok(c, gin.H{"subject": subject, "html": body, "copy_text": services.BuildCopyText(data)})
}

// ---------- 邮件模板 ----------

// GET /api/v1/mail/templates
func ListMailTemplates(c *gin.Context) {
	var list []models.SysMailTemplate
	models.DB.Order("id").Find(&list)
	ok(c, gin.H{"list": list})
}

type templatePayload struct {
	Name            string `json:"name"`
	SubjectTemplate string `json:"subject_template"`
	HTMLTemplate    string `json:"html_template"`
	IsDefault       *int8  `json:"is_default"`
	Status          *int8  `json:"status"`
}

// POST /api/v1/mail/templates
func CreateMailTemplate(c *gin.Context) {
	var p templatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Name == "" || p.SubjectTemplate == "" || p.HTMLTemplate == "" {
		fail(c, 400, "模板名称、标题模板、HTML 模板均不能为空")
		return
	}
	t := models.SysMailTemplate{Name: p.Name, SubjectTemplate: p.SubjectTemplate, HTMLTemplate: p.HTMLTemplate, Status: 1}
	if p.IsDefault != nil {
		t.IsDefault = *p.IsDefault
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	if err := models.DB.Create(&t).Error; err != nil {
		fail(c, 500, "创建模板失败: "+err.Error())
		return
	}
	if t.IsDefault == 1 {
		models.DB.Model(&models.SysMailTemplate{}).Where("id <> ?", t.ID).Update("is_default", 0)
	}
	okMsg(c, "模板创建成功", t)
}

// PUT /api/v1/mail/templates/:id
func UpdateMailTemplate(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var t models.SysMailTemplate
	if err := models.DB.First(&t, id).Error; err != nil {
		fail(c, 404, "模板不存在")
		return
	}
	var p templatePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Name != "" {
		t.Name = p.Name
	}
	if p.SubjectTemplate != "" {
		t.SubjectTemplate = p.SubjectTemplate
	}
	if p.HTMLTemplate != "" {
		t.HTMLTemplate = p.HTMLTemplate
	}
	if p.IsDefault != nil {
		t.IsDefault = *p.IsDefault
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	if err := models.DB.Save(&t).Error; err != nil {
		fail(c, 500, "保存模板失败")
		return
	}
	if t.IsDefault == 1 {
		models.DB.Model(&models.SysMailTemplate{}).Where("id <> ?", t.ID).Update("is_default", 0)
	}
	okMsg(c, "模板更新成功", nil)
}

// DELETE /api/v1/mail/templates/:id
func DeleteMailTemplate(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var t models.SysMailTemplate
	if err := models.DB.First(&t, id).Error; err != nil {
		fail(c, 404, "模板不存在")
		return
	}
	if t.IsDefault == 1 {
		fail(c, 400, "默认模板不可删除，请先将其他模板设为默认")
		return
	}
	models.DB.Delete(&t)
	okMsg(c, "模板已删除", nil)
}
