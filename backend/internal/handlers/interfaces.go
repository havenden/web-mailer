package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"web-mailer/internal/models"
)

type interfacePayload struct {
	Name                 string  `json:"name"`
	Region               string  `json:"region"`
	QualificationAddress string  `json:"qualification_address"`
	Channel              string  `json:"channel"`
	Remark               string  `json:"remark"`
	Status               *int8   `json:"status"`
	UserIDs              []int64 `json:"user_ids"`
}

func generateInterfaceKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GET /api/v1/interfaces
func ListInterfaces(c *gin.Context) {
	page, pageSize := pageParams(c)
	q := models.DB.Model(&models.SysInterface{})
	for _, f := range []struct{ key, col string }{
		{"name", "name"}, {"region", "region"}, {"channel", "channel"},
	} {
		if v := c.Query(f.key); v != "" {
			q = q.Where(f.col+" LIKE ?", "%"+v+"%")
		}
	}
	if s := c.Query("status"); s == "1" || s == "0" {
		q = q.Where("status = ?", s)
	}

	var total int64
	q.Count(&total)
	var list []models.SysInterface
	q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	if len(list) == 0 {
		ok(c, gin.H{"total": total, "list": []interface{}{}})
		return
	}

	ifaceIDs := make([]int64, len(list))
	for i, it := range list {
		ifaceIDs[i] = it.ID
	}

	// 批量查询绑定用户数
	type idCount struct {
		InterfaceID int64
		Cnt         int64
	}
	var userCounts []idCount
	models.DB.Model(&models.SysInterfaceUser{}).
		Select("interface_id, COUNT(*) as cnt").
		Where("interface_id IN ?", ifaceIDs).
		Group("interface_id").Scan(&userCounts)
	ucMap := make(map[int64]int64, len(userCounts))
	for _, uc := range userCounts {
		ucMap[uc.InterfaceID] = uc.Cnt
	}

	// 批量查询接收数量
	var pushCounts []idCount
	models.DB.Model(&models.SysPushRecord{}).
		Select("interface_id, COUNT(*) as cnt").
		Where("interface_id IN ?", ifaceIDs).
		Group("interface_id").Scan(&pushCounts)
	pcMap := make(map[int64]int64, len(pushCounts))
	for _, pc := range pushCounts {
		pcMap[pc.InterfaceID] = pc.Cnt
	}

	// 批量查询最近接收时间
	type lastPush struct {
		InterfaceID int64
		MaxCreated  time.Time
	}
	var lastPushes []lastPush
	models.DB.Model(&models.SysPushRecord{}).
		Select("interface_id, MAX(created_at) as max_created").
		Where("interface_id IN ?", ifaceIDs).
		Group("interface_id").Scan(&lastPushes)
	lpMap := make(map[int64]time.Time, len(lastPushes))
	for _, lp := range lastPushes {
		lpMap[lp.InterfaceID] = lp.MaxCreated
	}

	type ifaceItem struct {
		models.SysInterface
		UserCount  int64  `json:"user_count"`
		PushCount  int64  `json:"push_count"`
		LastPushAt string `json:"last_push_at"`
	}
	items := make([]ifaceItem, 0, len(list))
	for _, it := range list {
		item := ifaceItem{SysInterface: it, UserCount: ucMap[it.ID], PushCount: pcMap[it.ID]}
		if t, ok := lpMap[it.ID]; ok {
			s := t.Format("2006-01-02 15:04:05")
			item.LastPushAt = s
		}
		items = append(items, item)
	}
	ok(c, gin.H{"total": total, "list": items})
}

// GET /api/v1/interfaces/:id
func GetInterface(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var iface models.SysInterface
	if err := models.DB.Unscoped().First(&iface, id).Error; err != nil {
		fail(c, 404, "接口不存在")
		return
	}
	// 绑定用户
	var users []models.SysUser
	models.DB.Joins("JOIN sys_interface_users iu ON iu.user_id = sys_users.id").
		Where("iu.interface_id = ?", id).Find(&users)
	// 统计
	var pushCount int64
	models.DB.Model(&models.SysPushRecord{}).Where("interface_id = ?", id).Count(&pushCount)
	var mailStats []struct {
		Status int8
		Cnt    int64
	}
	models.DB.Model(&models.SysMailRecord{}).Select("status, count(*) as cnt").
		Where("interface_id = ?", id).Group("status").Scan(&mailStats)

	ok(c, gin.H{"interface": iface, "users": users, "push_count": pushCount, "mail_stats": mailStats})
}

// POST /api/v1/interfaces
func CreateInterface(c *gin.Context) {
	var p interfacePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Name == "" {
		fail(c, 400, "接口名称不能为空")
		return
	}
	key, err := generateInterfaceKey()
	if err != nil {
		fail(c, 500, "生成 interface_key 失败")
		return
	}
	status := int8(1)
	if p.Status != nil {
		status = *p.Status
	}
	iface := models.SysInterface{
		Name: p.Name, InterfaceKey: key, Region: p.Region,
		QualificationAddress: p.QualificationAddress, Channel: p.Channel,
		Remark: p.Remark, Status: status,
	}
	cfg := getAppConfig(c)
	iface.EndpointURL = fmt.Sprintf("%s/api/v1/push/%s", cfg.App.BaseURL, key)

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&iface).Error; err != nil {
			return err
		}
		return replaceInterfaceUsers(tx, iface.ID, p.UserIDs)
	})
	if err != nil {
		fail(c, 500, "创建接口失败")
		return
	}
	okMsg(c, "接口创建成功", iface)
}

// PUT /api/v1/interfaces/:id
func UpdateInterface(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var iface models.SysInterface
	if err := models.DB.Unscoped().First(&iface, id).Error; err != nil {
		fail(c, 404, "接口不存在")
		return
	}
	var p interfacePayload
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if p.Name != "" {
		iface.Name = p.Name
	}
	iface.Region = p.Region
	iface.QualificationAddress = p.QualificationAddress
	iface.Channel = p.Channel
	iface.Remark = p.Remark
	if p.Status != nil {
		iface.Status = *p.Status
	}
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&iface).Error; err != nil {
			return err
		}
		if p.UserIDs != nil {
			return replaceInterfaceUsers(tx, iface.ID, p.UserIDs)
		}
		return nil
	})
	if err != nil {
		fail(c, 500, "保存接口失败")
		return
	}
	okMsg(c, "接口更新成功", iface)
}

// PUT /api/v1/interfaces/:id/users 设置绑定用户
func SetInterfaceUsers(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var iface models.SysInterface
	if err := models.DB.First(&iface, id).Error; err != nil {
		fail(c, 404, "接口不存在")
		return
	}
	var p struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if err := models.DB.Transaction(func(tx *gorm.DB) error {
		return replaceInterfaceUsers(tx, id, p.UserIDs)
	}); err != nil {
		fail(c, 500, "绑定用户失败")
		return
	}
	okMsg(c, "绑定用户已更新", nil)
}

// DELETE /api/v1/interfaces/:id 逻辑删除
func DeleteInterface(c *gin.Context) {
	id := atoi64(c.Param("id"))
	if err := models.DB.Delete(&models.SysInterface{}, id).Error; err != nil {
		fail(c, 500, "删除接口失败")
		return
	}
	models.DB.Where("interface_id = ?", id).Delete(&models.SysInterfaceUser{})
	okMsg(c, "接口已删除", nil)
}

// replaceInterfaceUsers 全量替换绑定用户（校验用户存在且启用）
func replaceInterfaceUsers(tx *gorm.DB, interfaceID int64, userIDs []int64) error {
	if err := tx.Where("interface_id = ?", interfaceID).Delete(&models.SysInterfaceUser{}).Error; err != nil {
		return err
	}
	for _, uid := range userIDs {
		var u models.SysUser
		if err := tx.First(&u, uid).Error; err != nil {
			return fmt.Errorf("用户 #%d 不存在", uid)
		}
		if u.Status != 1 {
			return fmt.Errorf("用户「%s」已被禁用，不能绑定", u.Nickname)
		}
		if err := tx.Create(&models.SysInterfaceUser{InterfaceID: interfaceID, UserID: uid}).Error; err != nil {
			return err
		}
	}
	return nil
}
