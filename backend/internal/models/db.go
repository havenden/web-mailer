package models

import "gorm.io/gorm"

// DB 全局数据库实例（由 database.Init 初始化）
var DB *gorm.DB
