package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"web-mailer/internal/config"
	"web-mailer/internal/models"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// MakeToken 生成后台登录 JWT
func MakeToken(cfg *config.Config, u *models.SysUser) (string, time.Time, error) {
	expire := time.Now().Add(time.Duration(cfg.JWT.ExpireHours) * time.Hour)
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expire),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "web-mailer",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(cfg.JWT.Secret))
	return s, expire, err
}

// Auth 后台登录态校验（与第三方推送接口无关）
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录或登录已过期"})
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWT.Secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "登录已过期，请重新登录"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// CORS 跨域中间件（开发阶段前端 Vite 代理 + 生产同源部署，宽松配置）
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Vary", "Origin")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// LoginRateLimiter 登录接口频率限制（每 IP 每分钟最多 10 次）
type LoginRateLimiter struct {
	mu       sync.Mutex
	records  map[string]*loginRecord
}

type loginRecord struct {
	count  int
	window time.Time
}

func NewLoginRateLimiter() *LoginRateLimiter {
	return &LoginRateLimiter{records: make(map[string]*loginRecord)}
}

func (l *LoginRateLimiter) Allow(ip string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	rec, ok := l.records[ip]
	if !ok || now.Sub(rec.window) > window {
		l.records[ip] = &loginRecord{count: 1, window: now}
		return true
	}
	rec.count++
	return rec.count <= limit
}
