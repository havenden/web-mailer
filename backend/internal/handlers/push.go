package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"fmt"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/models"
	"web-mailer/internal/services"
)

// POST /api/v1/push/:interface_key 第三方数据接收入口（无鉴权、不去重）
func Push(c *gin.Context) {
	cfg := getAppConfig(c)
	key := c.Param("interface_key")

	// 1. 读取请求体（限制大小）
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, cfg.Push.MaxBodySizeBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "读取请求体失败"})
		return
	}
	if int64(len(body)) > cfg.Push.MaxBodySizeBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"code": 413, "message": "请求体超过大小限制"})
		return
	}

	// 2. 定位接口配置
	var iface models.SysInterface
	if err := models.DB.Where("interface_key = ? AND deleted_at IS NULL", key).First(&iface).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在或已被删除", "data": nil})
		return
	}
	if iface.Status != 1 {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "接口已禁用，不接受新数据", "data": nil})
		return
	}

	// 3. JSON 合法性校验 + 宽松解析（各渠道字段类型不一致：clueCreatetime 可能是时间戳数字等，统一转文本）
	if !json.Valid(body) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求体不是合法的 JSON", "data": nil})
		return
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(body, &payloadMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "JSON 解析失败: " + err.Error(), "data": nil})
		return
	}
	phone := services.PickString(payloadMap, "phone")
	name := services.PickString(payloadMap, "name")
	extra := services.ExtractPushExtra(string(body))
	if phone == "" {
		phone = extra["phone"] // 兼容 mobile / tel / 手机号 等命名
	}
	if name == "" {
		name = extra["name"]
	}
	clueSource := services.PickString(payloadMap, "clueSource")

	// 4. 业务校验
	if cfg.Push.RequirePhone && phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少必填字段 phone（客户手机号，兼容 mobile / 手机号 等命名）", "data": nil})
		return
	}

	// 5. 保存接收记录（每次有效请求均为新数据，不去重）
	// 渠道：接口配置优先，为空时回退 JSON clueSource
	channel := iface.Channel
	if channel == "" {
		channel = clueSource
	}
	rec := models.SysPushRecord{
		InterfaceID: iface.ID,
		Sequence:    fmt.Sprintf("%s%04d", time.Now().Format("20060102150405"), rand.Intn(10000)),
		RawJSON:     string(body),
		Phone:       phone,
		Name:        name,
		Channel:     channel,
	}
	if err := models.DB.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据保存失败，请稍后重试", "data": nil})
		return
	}

	// 6. 为每个绑定用户创建独立邮件任务（收件人互相独立，不 CC/BCC）
	var userIDs []int64
	models.DB.Model(&models.SysInterfaceUser{}).Where("interface_id = ?", iface.ID).Pluck("user_id", &userIDs)

	created := int64(0)
	for _, uid := range userIDs {
		var u models.SysUser
		// 仅启用用户接收新邮件任务
		if err := models.DB.Where("id = ? AND status = 1 AND deleted_at IS NULL", uid).First(&u).Error; err != nil {
			continue
		}
		mr := models.SysMailRecord{
			PushRecordID:   rec.ID,
			InterfaceID:    iface.ID,
			UserID:         u.ID,
			RecipientEmail: u.Email,
			RecipientName:  u.Nickname,
			Status:         models.MailStatusPending,
		}
		if models.DB.Create(&mr).Error == nil {
			created++
		}
	}
	if created == 0 {
		models.DB.Model(&rec).Update("process_note", "无可用收件用户")
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "数据接收成功",
		"data": gin.H{
			"record_id":      rec.ID,
			"sequence":       rec.Sequence,
			"mail_tasks":     created,
		},
	})
}
