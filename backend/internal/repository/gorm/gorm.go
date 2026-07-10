package gorm

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DBConfig struct {
	Driver string // "sqlite" | "postgres" | "mysql"
	DSN    string
}

func OpenDB(cfg DBConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN + "?_journal_mode=WAL&_busy_timeout=5000")
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := db.AutoMigrate(
		&History{},
		&StoryboardProject{}, &StoryboardShot{},
		&Setting{}, &Asset{}, &AccessLog{}, &TaskRecord{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}
	return db, nil
}
