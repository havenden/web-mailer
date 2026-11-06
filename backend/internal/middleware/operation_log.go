package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/models"
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// OperationLog 记录后台管理操作（跳过查询类请求与登录路由）
func OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if strings.Contains(path, "/auth/") {
			c.Next()
			return
		}

		bw := &bodyCaptureWriter{ResponseWriter: c.Writer, body: make([]byte, 0, 256)}
		c.Writer = bw

		c.Next()

		result := "success"
		if c.Writer.Status() >= 400 {
			result = "fail"
		}

		var msg string
		respMap := map[string]interface{}{}
		if len(bw.body) > 0 && len(bw.body) < 4096 {
			_ = json.Unmarshal(bw.body, &respMap)
			msg, _ = respMap["message"].(string)
		}

		var op models.SysOperationLog
		if v, ok := c.Get("user_id"); ok {
			op.OperatorID, _ = v.(int64)
		}
		if v, ok := c.Get("username"); ok {
			op.Operator, _ = v.(string)
		}
		op.Module = guessModule(path)
		op.Action = c.Request.Method
		op.Description = truncateStr(msg, 500)
		op.Result = result
		op.SourceIP = c.ClientIP()
		models.DB.Create(&op)
	}
}

func guessModule(path string) string {
	switch {
	case strings.Contains(path, "/users"):
		return "用户管理"
	case strings.Contains(path, "/interfaces"):
		return "接口管理"
	case strings.Contains(path, "/mail/configs"):
		return "邮件配置"
	case strings.Contains(path, "/mail/templates"):
		return "邮件模板"
	case strings.Contains(path, "/mail-records"):
		return "邮件记录"
	case strings.Contains(path, "/push-records"):
		return "接收记录"
	}
	return "系统"
}

func truncateStr(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n]) + "..."
	}
	return string(r)
}
