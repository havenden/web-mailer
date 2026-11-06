package handlers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/models"
	"web-mailer/internal/services"
)

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type userPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Status   *int8  `json:"status"`
}

// GET /api/v1/users
func ListUsers(c *gin.Context) {
	page, pageSize := pageParams(c)
	kw := c.Query("keyword")
	status := c.Query("status")

	q := models.DB.Model(&models.SysUser{})
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}
	if status == "1" || status == "0" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	type userItem struct {
		models.SysUser
		InterfaceCount int64 `json:"interface_count"`
	}
	var items []userItem
	q.Select("sys_users.*, (SELECT COUNT(*) FROM sys_interface_users WHERE user_id = sys_users.id) AS interface_count").
		Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items)

	ok(c, gin.H{"total": total, "list": items})
}

// GET /api/v1/users/:id
func GetUser(c *gin.Context) {
	var user models.SysUser
	if err := models.DB.First(&user, atoi64(c.Param("id"))).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	ok(c, user)
}

// POST /api/v1/users
func CreateUser(c *gin.Context) {
	var p userPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Username == "" {
		fail(c, 400, "用户名不能为空")
		return
	}
	if p.Email == "" || !emailRe.MatchString(p.Email) {
		fail(c, 400, "邮箱格式不正确")
		return
	}
	if p.Password == "" {
		p.Password = "123456" // 默认初始密码
	}
	var cnt int64
	models.DB.Model(&models.SysUser{}).Where("username = ?", p.Username).Count(&cnt)
	if cnt > 0 {
		fail(c, 400, "用户名已存在")
		return
	}
	hash, err := services.HashPassword(p.Password)
	if err != nil {
		fail(c, 500, "密码加密失败")
		return
	}
	role := p.Role
	if role != "admin" && role != "user" {
		role = "user"
	}
	status := int8(1)
	if p.Status != nil {
		status = *p.Status
	}
	u := models.SysUser{
		Username: p.Username, PasswordHash: hash, Nickname: p.Nickname,
		Email: p.Email, Phone: p.Phone, Role: role, Status: status,
	}
	if err := models.DB.Create(&u).Error; err != nil {
		fail(c, 500, "创建用户失败")
		return
	}
	okMsg(c, "用户创建成功", u)
}

// PUT /api/v1/users/:id
func UpdateUser(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var user models.SysUser
	if err := models.DB.First(&user, id).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	var p userPayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Email != "" && !emailRe.MatchString(p.Email) {
		fail(c, 400, "邮箱格式不正确")
		return
	}
	if p.Username != "" && p.Username != user.Username {
		var cnt int64
		models.DB.Model(&models.SysUser{}).Where("username = ? AND id <> ?", p.Username, id).Count(&cnt)
		if cnt > 0 {
			fail(c, 400, "用户名已存在")
			return
		}
		user.Username = p.Username
	}
	if p.Nickname != "" {
		user.Nickname = p.Nickname
	}
	if p.Email != "" {
		user.Email = p.Email
	}
	user.Phone = p.Phone
	if p.Role == "admin" || p.Role == "user" {
		user.Role = p.Role
	}
	if p.Status != nil {
		user.Status = *p.Status
	}
	if p.Password != "" {
		hash, err := services.HashPassword(p.Password)
		if err != nil {
			fail(c, 500, "密码加密失败")
			return
		}
		user.PasswordHash = hash
	}
	if err := models.DB.Save(&user).Error; err != nil {
		fail(c, 500, "保存用户失败")
		return
	}
	okMsg(c, "用户更新成功", user)
}

// DELETE /api/v1/users/:id
func DeleteUser(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var cnt int64
	models.DB.Model(&models.SysInterfaceUser{}).Where("user_id = ?", id).Count(&cnt)
	if cnt > 0 {
		var names []string
		models.DB.Model(&models.SysInterface{}).
			Joins("JOIN sys_interface_users iu ON iu.interface_id = sys_interfaces.id").
			Where("iu.user_id = ?", id).Pluck("name", &names)
		fail(c, 400, "该用户已绑定 "+strconv.FormatInt(cnt, 10)+" 个接口（"+joinNames(names)+"），请先解除绑定后再删除")
		return
	}
	if err := models.DB.Delete(&models.SysUser{}, id).Error; err != nil {
		fail(c, 500, "删除用户失败")
		return
	}
	okMsg(c, "用户已删除", nil)
}

// PUT /api/v1/users/:id/status 启用/禁用
func SetUserStatus(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var p struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if err := models.DB.Model(&models.SysUser{}).Where("id = ?", id).Update("status", p.Status).Error; err != nil {
		fail(c, 500, "操作失败")
		return
	}
	msg := "用户已禁用（不再接收新邮件任务）"
	if p.Status == 1 {
		msg = "用户已启用"
	}
	okMsg(c, msg, nil)
}

func joinNames(names []string) string {
	return strings.Join(names, "、")
}
