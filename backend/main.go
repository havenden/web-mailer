package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"web-mailer/internal/config"
	"web-mailer/internal/database"
	"web-mailer/internal/handlers"
	"web-mailer/internal/middleware"
	"web-mailer/internal/services"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := database.Init(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	r.MaxMultipartMemory = 8 << 20

	// 注入配置
	withCfg := func(c *gin.Context) {
		c.Set("appcfg", cfg)
		c.Next()
	}

	// 初始化登录频率限制
	handlers.LoginRateLimiter = middleware.NewLoginRateLimiter()

	// ---------- 第三方推送入口（无 JWT / API-Key） ----------
	pushGroup := r.Group("/api/v1/push", withCfg)
	{
		pushGroup.POST("/:interface_key", handlers.Push)
	}

	// ---------- 后台管理 API（JWT） ----------
	api := r.Group("/api/v1", withCfg, middleware.OperationLog())
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/logout", handlers.Logout)
		}
		// 以下需要登录
		authed := api.Group("", middleware.Auth(cfg))
		{
			authed.GET("/auth/info", handlers.AuthInfo)
			authed.PUT("/auth/password", handlers.ChangePassword)

			authed.GET("/users", handlers.ListUsers)
			authed.POST("/users", handlers.CreateUser)
			authed.GET("/users/:id", handlers.GetUser)
			authed.PUT("/users/:id", handlers.UpdateUser)
			authed.DELETE("/users/:id", handlers.DeleteUser)
			authed.PUT("/users/:id/status", handlers.SetUserStatus)

			authed.GET("/interfaces", handlers.ListInterfaces)
			authed.POST("/interfaces", handlers.CreateInterface)
			authed.GET("/interfaces/:id", handlers.GetInterface)
			authed.PUT("/interfaces/:id", handlers.UpdateInterface)
			authed.PUT("/interfaces/:id/users", handlers.SetInterfaceUsers)
			authed.DELETE("/interfaces/:id", handlers.DeleteInterface)

			authed.GET("/mail/configs", handlers.ListMailConfigs)
			authed.POST("/mail/configs", handlers.CreateMailConfig)
			authed.PUT("/mail/configs/:id", handlers.UpdateMailConfig)
			authed.DELETE("/mail/configs/:id", handlers.DeleteMailConfig)
			authed.POST("/mail/test", handlers.SendTestMail)
			authed.POST("/mail/preview", handlers.PreviewTemplate)

			authed.GET("/mail/templates", handlers.ListMailTemplates)
			authed.POST("/mail/templates", handlers.CreateMailTemplate)
			authed.PUT("/mail/templates/:id", handlers.UpdateMailTemplate)
			authed.DELETE("/mail/templates/:id", handlers.DeleteMailTemplate)

			authed.GET("/push-records", handlers.ListPushRecords)
			authed.GET("/push-records/:id", handlers.GetPushRecord)
			authed.DELETE("/push-records/:id", handlers.DeletePushRecord)
			authed.GET("/mail-records", handlers.ListMailRecords)
			authed.GET("/mail-records/:id", handlers.GetMailRecord)
			authed.GET("/mail-records/:id/copy-text", handlers.GetMailRecordCopyText)
			authed.POST("/mail-records/:id/retry", handlers.RetryMailRecord)
			authed.DELETE("/mail-records/:id", handlers.DeleteMailRecord)

			authed.GET("/dashboard", handlers.Dashboard)
			authed.GET("/logs", handlers.ListOperationLogs)
			authed.GET("/login-logs", handlers.ListLoginLogs)
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// ---------- 前端静态托管（frontend/dist 存在时启用单端口部署） ----------
	if st, err := os.Stat("../frontend/dist/index.html"); err == nil && st != nil {
		r.Static("/assets", "../frontend/dist/assets")
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
				return
			}
			c.File("../frontend/dist/index.html")
		})
		log.Println("已启用前端静态托管（../frontend/dist）")
	} else if st, err := os.Stat("frontend/dist/index.html"); err == nil && st != nil {
		r.Static("/assets", "frontend/dist/assets")
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
				return
			}
			c.File("frontend/dist/index.html")
		})
		log.Println("已启用前端静态托管（frontend/dist）")
	}

	// ---------- 异步邮件 Worker ----------
	worker := services.NewWorker(cfg)
	worker.Start()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		log.Printf("Web-Mailer 后端已启动: http://127.0.0.1:%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务启动失败: %v", err)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")
	worker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("服务已退出")
}
