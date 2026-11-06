package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/config"
)

func getAppConfig(c *gin.Context) *config.Config {
	if v, ok := c.Get("appcfg"); ok {
		if cfg, ok := v.(*config.Config); ok {
			return cfg
		}
	}
	return &config.Config{}
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func okMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": msg, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg})
}

func pageParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}

func atoi64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
