package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const databasePath = "./database/heybox.db"

var (
	database *gorm.DB
	dbMu     sync.Mutex
)

// Open 打开本地 SQLite 数据库，并返回全局 GORM 实例。
func Open() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if database != nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := initMessage(db); err != nil {
		return err
	}

	database = db
	return nil
}

// Close 关闭当前打开的数据库连接。
func Close() error {
	dbMu.Lock()
	defer dbMu.Unlock()

	if database == nil {
		return nil
	}

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}

	database = nil
	return nil
}

func getDatabase() (*gorm.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if database == nil {
		return nil, fmt.Errorf("数据库未打开")
	}
	return database, nil
}
