package services

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"context"

	"web-mailer/internal/models"
)

// MailData 邮件模板渲染数据
type MailData struct {
	Region               string
	Sequence             string
	Channel              string
	Phone                string
	Condition            string
	Nickname             string
	Doctor               string
	QualificationAddress string
	InterfaceName        string // 接口名称（邮件标题使用）
	RawJSONHTML          string // 已转义、换行已转 <br> 的原始 JSON（正文底部「完整数据」块）
}

// RenderSubject 渲染邮件标题
func RenderSubject(subjectTpl string, data *MailData) (string, error) {
	t, err := template.New("subject").Parse(subjectTpl)
	if err != nil {
		return "", fmt.Errorf("标题模板解析失败: %w", err)
	}
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// RenderHTML 渲染邮件正文（动态文本 HTML 转义，防注入）
func RenderHTML(htmlTpl string, data *MailData) (string, error) {
	// text/template 不会自动转义，这里手动对动态字段转义后再渲染
	escaped := &MailData{
		Region:               html.EscapeString(data.Region),
		Sequence:             html.EscapeString(data.Sequence),
		Channel:              html.EscapeString(data.Channel),
		Phone:                html.EscapeString(data.Phone),
		Condition:            html.EscapeString(data.Condition),
		Nickname:             html.EscapeString(data.Nickname),
		Doctor:               html.EscapeString(data.Doctor),
		QualificationAddress: html.EscapeString(data.QualificationAddress),
		InterfaceName:        html.EscapeString(data.InterfaceName),
		RawJSONHTML:          data.RawJSONHTML, // 内容本身已转义，直接透传
	}
	t, err := template.New("html").Parse(htmlTpl)
	if err != nil {
		return "", fmt.Errorf("正文模板解析失败: %w", err)
	}
	var sb strings.Builder
	if err := t.Execute(&sb, escaped); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// SendMail 通过 SMTP（QQ 邮箱 SSL 465 / STARTTLS 587）发送单封邮件
func SendMail(mc *models.SysMailConfig, authCode, to, subject, htmlContent string) error {
	if mc.SMTPHost == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	if authCode == "" {
		return fmt.Errorf("SMTP 授权码未配置")
	}

	from := mc.SenderEmail
	addr := net.JoinHostPort(mc.SMTPHost, strconv.Itoa(mc.SMTPPort))

	// 邮件头
	boundaryHeaders := buildMessage(from, mc.SenderName, to, subject, htmlContent)

	auth := smtp.PlainAuth("", mc.SMTPUsername, authCode, mc.SMTPHost)

	if mc.Encryption == "starttls" {
		// 587 STARTTLS
		conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
		if err != nil {
			return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
		}
		cl, err := smtp.NewClient(conn, mc.SMTPHost)
		if err != nil {
			return fmt.Errorf("建立 SMTP 会话失败: %w", err)
		}
		defer cl.Close()
		if err := cl.Hello(mc.SMTPHost); err != nil {
			return fmt.Errorf("SMTP HELO 失败: %w", err)
		}
		if ok, _ := cl.Extension("STARTTLS"); !ok {
			return fmt.Errorf("服务器不支持 STARTTLS")
		}
		tlsCfg := &tls.Config{ServerName: mc.SMTPHost}
		if err := cl.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("STARTTLS 失败: %w", err)
		}
		if err := cl.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败（请检查邮箱与授权码）: %w", err)
		}
		if err := cl.Mail(from); err != nil {
			return fmt.Errorf("设置发件人失败: %w", err)
		}
		if err := cl.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
		w, err := cl.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(boundaryHeaders)); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return cl.Quit()
	}

	// 默认 SSL 465
	tlsCfg := &tls.Config{ServerName: mc.SMTPHost}
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	rawConn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败（%s）: %w", addr, err)
	}
	conn := tls.Client(rawConn, tlsCfg)
	if err := conn.HandshakeContext(context.Background()); err != nil {
		rawConn.Close()
		return fmt.Errorf("SSL 握手失败（%s）: %w", addr, err)
	}
	cl, err := smtp.NewClient(conn, mc.SMTPHost)
	if err != nil {
		return fmt.Errorf("建立 SMTP 会话失败: %w", err)
	}
	defer cl.Close()
	if err := cl.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败（请检查邮箱与授权码）: %w", err)
	}
	if err := cl.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	if err := cl.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}
	w, err := cl.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(boundaryHeaders)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return cl.Quit()
}

func encodeDisplayName(name string) string {
	if name == "" {
		return ""
	}
	return mime.QEncoding.Encode("utf-8", name)
}

func buildMessage(from, fromName, to, subject, htmlContent string) string {
	msg := &strings.Builder{}
	msg.WriteString("From: " + encodeDisplayName(fromName) + " <" + from + ">\r\n")
	msg.WriteString("To: <" + to + ">\r\n")
	msg.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(htmlContent))
	// 按 76 字符换行（RFC 2045）
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		msg.WriteString(enc[i:end] + "\r\n")
	}
	return msg.String()
}
