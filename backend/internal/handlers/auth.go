package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/middleware"
	"web-mailer/internal/models"
	"web-mailer/internal/services"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginRateLimiter 由 main.go 初始化注入
var LoginRateLimiter *middleware.LoginRateLimiter

// POST /api/v1/auth/login
func Login(c *gin.Context) {
	if LoginRateLimiter != nil && !LoginRateLimiter.Allow(c.ClientIP(), 10, time.Minute) {
		fail(c, 429, "登录尝试过于频繁，请稍后再试")
		return
	}

	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "请输入用户名和密码")
		return
	}

	var user models.SysUser
	logRec := models.SysLoginLog{Username: req.Username, SourceIP: c.ClientIP()}
	if err := models.DB.Where("username = ? AND deleted_at IS NULL", req.Username).First(&user).Error; err != nil ||
		!services.CheckPassword(user.PasswordHash, req.Password) {
		logRec.LoginResult = "fail"
		logRec.Message = "用户名或密码错误"
		models.DB.Create(&logRec)
		fail(c, 401, "用户名或密码错误")
		return
	}
	if user.Status != 1 {
		logRec.LoginResult = "fail"
		logRec.Message = "账号已被禁用"
		models.DB.Create(&logRec)
		fail(c, 403, "账号已被禁用，请联系管理员")
		return
	}

	appCfg := getAppConfig(c)
	token, expire, err := middleware.MakeToken(appCfg, &user)
	if err != nil {
		fail(c, 500, "生成登录凭证失败")
		return
	}

	logRec.UserID = user.ID
	logRec.LoginResult = "success"
	models.DB.Create(&logRec)

	ok(c, gin.H{"token": token, "expires_at": expire, "user": gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname, "role": user.Role,
	}})
}

// POST /api/v1/auth/logout
func Logout(c *gin.Context) {
	okMsg(c, "已退出登录", nil)
}

// GET /api/v1/auth/info
func AuthInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var user models.SysUser
	if err := models.DB.First(&user, userID).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	ok(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"role": user.Role, "email": user.Email,
	})
}

// PUT /api/v1/auth/password 修改当前登录用户密码
type changePwdReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	var req changePwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误（新密码至少 6 位）")
		return
	}
	userID, _ := c.Get("user_id")
	var user models.SysUser
	if err := models.DB.First(&user, userID).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	if !services.CheckPassword(user.PasswordHash, req.OldPassword) {
		fail(c, 400, "原密码错误")
		return
	}
	hash, err := services.HashPassword(req.NewPassword)
	if err != nil {
		fail(c, 500, "密码加密失败")
		return
	}
	models.DB.Model(&user).Update("password_hash", hash)
	okMsg(c, "密码修改成功", nil)
}
