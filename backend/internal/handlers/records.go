package handlers

import (
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/models"
	"web-mailer/internal/services"
)

// GET /api/v1/push-records 数据接收记录查询
func ListPushRecords(c *gin.Context) {
	page, pageSize := pageParams(c)
	q := models.DB.Model(&models.SysPushRecord{})
	if ifaceID := c.Query("interface_id"); ifaceID != "" {
		q = q.Where("interface_id = ?", atoi64(ifaceID))
	}
	if phone := c.Query("phone"); phone != "" {
		q = q.Where("phone LIKE ?", "%"+phone+"%")
	}
	if seq := c.Query("sequence"); seq != "" {
		q = q.Where("sequence LIKE ?", "%"+seq+"%")
	}
	if start := c.Query("start_time"); start != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", start, time.Local); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", end, time.Local); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}

	var total int64
	q.Count(&total)
	var list []models.SysPushRecord
	q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	if len(list) == 0 {
		ok(c, gin.H{"total": total, "list": []interface{}{}})
		return
	}

	ifaceIDs := make([]int64, 0, len(list))
	for _, r := range list {
		ifaceIDs = append(ifaceIDs, r.InterfaceID)
	}
	var ifaces []models.SysInterface
	models.DB.Unscoped().Where("id IN ?", ifaceIDs).Find(&ifaces)
	ifaceMap := make(map[int64]string, len(ifaces))
	for _, f := range ifaces {
		ifaceMap[f.ID] = f.Name
	}

	type mailStatRow struct {
		PushRecordID int64
		Total        int64
		Success      int64
		Failed       int64
		Pending      int64
	}
	recordIDs := make([]int64, len(list))
	for i, r := range list {
		recordIDs[i] = r.ID
	}
	var stats []mailStatRow
	models.DB.Model(&models.SysMailRecord{}).
		Select("push_record_id, COUNT(*) as total, "+
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as success, "+
			"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as failed, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as pending",
			models.MailStatusSuccess, models.MailStatusFailed,
			[]int8{models.MailStatusPending, models.MailStatusRetry, models.MailStatusSending}).
		Where("push_record_id IN ?", recordIDs).
		Group("push_record_id").Scan(&stats)

	statMap := make(map[int64]mailStatRow, len(stats))
	for _, s := range stats {
		statMap[s.PushRecordID] = s
	}

	type pushItem struct {
		models.SysPushRecord
		InterfaceName string `json:"interface_name"`
		MailTotal     int64  `json:"mail_total"`
		MailSuccess   int64  `json:"mail_success"`
		MailFailed    int64  `json:"mail_failed"`
		MailPending   int64  `json:"mail_pending"`
	}
	items := make([]pushItem, 0, len(list))
	for _, r := range list {
		item := pushItem{SysPushRecord: r, InterfaceName: ifaceMap[r.InterfaceID]}
		if s, ok := statMap[r.ID]; ok {
			item.MailTotal = s.Total
			item.MailSuccess = s.Success
			item.MailFailed = s.Failed
			item.MailPending = s.Pending
		}
		items = append(items, item)
	}
	ok(c, gin.H{"total": total, "list": items})
}

// GET /api/v1/push-records/:id 记录详情及原始 JSON
func GetPushRecord(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var rec models.SysPushRecord
	if err := models.DB.First(&rec, id).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}
	var iface models.SysInterface
	ifaceName := ""
	if err := models.DB.Unscoped().First(&iface, rec.InterfaceID).Error; err == nil {
		ifaceName = iface.Name
	}
	var mails []models.SysMailRecord
	models.DB.Where("push_record_id = ?", id).Order("id").Find(&mails)
	ok(c, gin.H{"record": rec, "interface_name": ifaceName, "mail_records": mails})
}

// GET /api/v1/mail-records 邮件发送记录查询
func ListMailRecords(c *gin.Context) {
	page, pageSize := pageParams(c)
	q := models.DB.Model(&models.SysMailRecord{})
	if userID := c.Query("user_id"); userID != "" {
		q = q.Where("user_id = ?", atoi64(userID))
	}
	if email := c.Query("email"); email != "" {
		q = q.Where("recipient_email LIKE ?", "%"+email+"%")
	}
	if ifaceID := c.Query("interface_id"); ifaceID != "" {
		q = q.Where("interface_id = ?", atoi64(ifaceID))
	}
	if pushID := c.Query("push_record_id"); pushID != "" {
		q = q.Where("push_record_id = ?", atoi64(pushID))
	}
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", atoi64(s))
	}
	if start := c.Query("start_time"); start != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", start, time.Local); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", end, time.Local); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}

	var total int64
	q.Count(&total)
	var list []models.SysMailRecord
	q.Select("id, push_record_id, interface_id, user_id, recipient_email, recipient_name, subject, status, attempt_count, last_error, last_sent_at, next_retry_at, sent_at, created_at, updated_at").
		Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	if len(list) == 0 {
		ok(c, gin.H{"total": total, "list": []interface{}{}})
		return
	}

	userIDs := make([]int64, 0, len(list))
	for _, r := range list {
		userIDs = append(userIDs, r.UserID)
	}
	var users []models.SysUser
	models.DB.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[int64]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	ifaceIDs := make([]int64, 0, len(list))
	for _, r := range list {
		ifaceIDs = append(ifaceIDs, r.InterfaceID)
	}
	var ifaces []models.SysInterface
	models.DB.Unscoped().Where("id IN ?", ifaceIDs).Find(&ifaces)
	ifaceMap := make(map[int64]string, len(ifaces))
	for _, f := range ifaces {
		ifaceMap[f.ID] = f.Name
	}

	type mailItem struct {
		models.SysMailRecord
		Username      string `json:"username"`
		InterfaceName string `json:"interface_name"`
	}
	items := make([]mailItem, 0, len(list))
	for _, r := range list {
		item := mailItem{
			SysMailRecord: r,
			Username:      userMap[r.UserID],
			InterfaceName: ifaceMap[r.InterfaceID],
		}
		items = append(items, item)
	}
	ok(c, gin.H{"total": total, "list": items})
}

// GET /api/v1/mail-records/:id
func GetMailRecord(c *gin.Context) {
	var rec models.SysMailRecord
	if err := models.DB.First(&rec, atoi64(c.Param("id"))).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}
	copyText, _ := services.MailCopyText(&rec)
	ok(c, gin.H{
		"id": rec.ID, "push_record_id": rec.PushRecordID, "interface_id": rec.InterfaceID,
		"user_id": rec.UserID, "recipient_email": rec.RecipientEmail, "recipient_name": rec.RecipientName,
		"subject": rec.Subject, "html_content": rec.HTMLContent, "status": rec.Status,
		"attempt_count": rec.AttemptCount, "last_error": rec.LastError,
		"last_sent_at": rec.LastSentAt, "sent_at": rec.SentAt, "created_at": rec.CreatedAt,
		"copy_text": copyText,
	})
}

// GET /api/v1/mail-records/:id/copy-text 生成「序号 → 资质所属地」纯文本（供一键复制）
func GetMailRecordCopyText(c *gin.Context) {
	var rec models.SysMailRecord
	if err := models.DB.First(&rec, atoi64(c.Param("id"))).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}
	text, err := services.MailCopyText(&rec)
	if err != nil {
		fail(c, 400, "生成复制文本失败: "+err.Error())
		return
	}
	ok(c, gin.H{"text": text})
}

// POST /api/v1/mail-records/:id/retry 手动重发
func RetryMailRecord(c *gin.Context) {
	id := atoi64(c.Param("id"))
	if err := services.RetryMail(id); err != nil {
		fail(c, 400, err.Error())
		return
	}
	okMsg(c, "已重置为待发送，Worker 将尽快处理", nil)
}

// GET /api/v1/dashboard 仪表盘统计
func Dashboard(c *gin.Context) {
	var userTotal, userActive, ifaceTotal, ifaceActive int64
	models.DB.Model(&models.SysUser{}).Count(&userTotal)
	models.DB.Model(&models.SysUser{}).Where("status = 1").Count(&userActive)
	models.DB.Model(&models.SysInterface{}).Count(&ifaceTotal)
	models.DB.Model(&models.SysInterface{}).Where("status = 1").Count(&ifaceActive)

	var pushToday, pushTotal int64
	todayStart := time.Now().Format("2006-01-02") + " 00:00:00"
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", todayStart, time.Local)
	models.DB.Model(&models.SysPushRecord{}).Where("created_at >= ?", t).Count(&pushToday)
	models.DB.Model(&models.SysPushRecord{}).Count(&pushTotal)

	var mailToday, mailSuccess, mailFailed, mailPending int64
	models.DB.Model(&models.SysMailRecord{}).Where("created_at >= ?", t).Count(&mailToday)
	models.DB.Model(&models.SysMailRecord{}).Where("status = ?", models.MailStatusSuccess).Count(&mailSuccess)
	models.DB.Model(&models.SysMailRecord{}).Where("status = ?", models.MailStatusFailed).Count(&mailFailed)
	models.DB.Model(&models.SysMailRecord{}).Where("status IN ?", []int8{models.MailStatusPending, models.MailStatusRetry, models.MailStatusSending}).Count(&mailPending)

	var mailTotal int64
	models.DB.Model(&models.SysMailRecord{}).Count(&mailTotal)
	successRate := 0.0
	if mailTotal > 0 {
		successRate = float64(mailSuccess) / float64(mailTotal) * 100
	}

	// 最近 7 天趋势 — 批量 GROUP BY 替代逐天查询
	type TrendPoint struct {
		Date       string `json:"date"`
		PushCount  int64  `json:"push_count"`
		MailCount  int64  `json:"mail_count"`
	}
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -6)
	sevenDaysAgoStart := time.Date(sevenDaysAgo.Year(), sevenDaysAgo.Month(), sevenDaysAgo.Day(), 0, 0, 0, 0, now.Location())

	type dateCount struct {
		Date string
		Cnt  int64
	}
	var pushTrend []dateCount
	models.DB.Model(&models.SysPushRecord{}).
		Select("DATE(created_at) as date, COUNT(*) as cnt").
		Where("created_at >= ?", sevenDaysAgoStart).
		Group("DATE(created_at)").Scan(&pushTrend)
	pushMap := make(map[string]int64, len(pushTrend))
	for _, p := range pushTrend {
		pushMap[p.Date] = p.Cnt
	}

	var mailTrend []dateCount
	models.DB.Model(&models.SysMailRecord{}).
		Select("DATE(created_at) as date, COUNT(*) as cnt").
		Where("created_at >= ?", sevenDaysAgoStart).
		Group("DATE(created_at)").Scan(&mailTrend)
	mailMap := make(map[string]int64, len(mailTrend))
	for _, m := range mailTrend {
		mailMap[m.Date] = m.Cnt
	}

	trend := make([]TrendPoint, 0, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		ds := day.Format("2006-01-02")
		trend = append(trend, TrendPoint{
			Date:      day.Format("01-02"),
			PushCount: pushMap[ds],
			MailCount: mailMap[ds],
		})
	}

	// 按接口排行 — 批量查询替代逐接口查询
	type IfaceRank struct {
		InterfaceID   int64  `json:"interface_id"`
		InterfaceName string `json:"interface_name"`
		PushCount     int64  `json:"push_count"`
		MailCount     int64  `json:"mail_count"`
	}
	var ifaces []models.SysInterface
	models.DB.Order("id desc").Limit(50).Find(&ifaces)

	rank := make([]IfaceRank, 0)
	if len(ifaces) > 0 {
		ifaceIDs := make([]int64, len(ifaces))
		ifaceNameMap := make(map[int64]string, len(ifaces))
		for i, f := range ifaces {
			ifaceIDs[i] = f.ID
			ifaceNameMap[f.ID] = f.Name
		}

		type idCount struct {
			ID  int64
			Cnt int64
		}
		var pushRanks []idCount
		models.DB.Model(&models.SysPushRecord{}).
			Select("interface_id as id, COUNT(*) as cnt").
			Where("interface_id IN ?", ifaceIDs).
			Group("interface_id").Scan(&pushRanks)

		rankMap := make(map[int64]*IfaceRank, len(ifaces))
		for _, pr := range pushRanks {
			rankMap[pr.ID] = &IfaceRank{InterfaceID: pr.ID, InterfaceName: ifaceNameMap[pr.ID], PushCount: pr.Cnt}
		}

		var mailRanks []idCount
		models.DB.Model(&models.SysMailRecord{}).
			Select("interface_id as id, COUNT(*) as cnt").
			Where("interface_id IN ?", ifaceIDs).
			Group("interface_id").Scan(&mailRanks)
		for _, mr := range mailRanks {
			if r, ok := rankMap[mr.ID]; ok {
				r.MailCount = mr.Cnt
			} else {
				rankMap[mr.ID] = &IfaceRank{InterfaceID: mr.ID, InterfaceName: ifaceNameMap[mr.ID], MailCount: mr.Cnt}
			}
		}

		for _, r := range rankMap {
			if r.PushCount > 0 || r.MailCount > 0 {
				rank = append(rank, *r)
			}
		}
		sort.Slice(rank, func(i, j int) bool { return rank[i].PushCount > rank[j].PushCount })
	}

	ok(c, gin.H{
		"user_total": userTotal, "user_active": userActive,
		"interface_total": ifaceTotal, "interface_active": ifaceActive,
		"push_today": pushToday, "push_total": pushTotal,
		"mail_today": mailToday, "mail_success": mailSuccess,
		"mail_failed": mailFailed, "mail_pending": mailPending,
		"mail_success_rate": successRate,
		"trend":             trend,
		"rank":              rank,
	})
}

// GET /api/v1/logs 操作日志
func ListOperationLogs(c *gin.Context) {
	page, pageSize := pageParams(c)
	q := models.DB.Model(&models.SysOperationLog{})
	if module := c.Query("module"); module != "" {
		q = q.Where("module = ?", module)
	}
	var total int64
	q.Count(&total)
	var list []models.SysOperationLog
	q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	ok(c, gin.H{"total": total, "list": list})
}

// GET /api/v1/login-logs 登录日志
func ListLoginLogs(c *gin.Context) {
	page, pageSize := pageParams(c)
	var total int64
	models.DB.Model(&models.SysLoginLog{}).Count(&total)
	var list []models.SysLoginLog
	models.DB.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)
	ok(c, gin.H{"total": total, "list": list})
}

// DELETE /api/v1/push-records/:id 删除接收记录（级联删除关联邮件）
func DeletePushRecord(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var rec models.SysPushRecord
	if err := models.DB.First(&rec, id).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}
	models.DB.Where("push_record_id = ?", id).Delete(&models.SysMailRecord{})
	models.DB.Delete(&rec)
	okMsg(c, "接收记录已删除", nil)
}

// DELETE /api/v1/mail-records/:id 删除邮件记录
func DeleteMailRecord(c *gin.Context) {
	id := atoi64(c.Param("id"))
	var rec models.SysMailRecord
	if err := models.DB.First(&rec, id).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}
	models.DB.Delete(&rec)
	okMsg(c, "邮件记录已删除", nil)
}
