package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"

	"userclient/internal/config"
	"userclient/internal/models"
)

// DB 数据库实例
type DB struct {
	*gorm.DB
}

// New 创建数据库连接
func New(cfg *config.DatabaseConfig) (*DB, error) {
	// 确保数据目录存在
	dataDir := filepath.Dir(cfg.DSN)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 配置GORM日志级别
	logLevel := getLogLevel(cfg.LogLevel)

	// 打开数据库连接（使用modernc.org/sqlite驱动）
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        cfg.DSN,
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层sql.DB实例进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logrus.Info("数据库连接成功")

	return &DB{DB: db}, nil
}

// AutoMigrate 自动迁移数据库表
func (db *DB) AutoMigrate() error {
	logrus.Info("开始数据库迁移...")

	// 迁移所有模型
	err := db.DB.AutoMigrate(
		&models.BarcodeRecord{},
		&models.Device{},
		&models.Configuration{},
		&models.SystemLog{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	logrus.Info("数据库迁移完成")
	return nil
}

// seedDevices 初始化设备数据
func (db *DB) seedDevices() error {
	// 检查是否已存在设备
	var count int64
	db.Model(&models.Device{}).Count(&count)
	if count > 0 {
		return nil // 已存在设备，跳过初始化
	}

	// 创建默认设备
	defaultDevice := models.Device{
		Name:        "默认扫码枪",
		Type:        "scanner",
		Model:       "Generic USB Scanner",
		SerialNo:    "DEFAULT-001",
		Description: "系统默认扫码枪设备",
		Status:      "active",
		IsActive:    true,
	}

	return db.Create(&defaultDevice).Error
}

// seedConfigurations 初始化系统配置
func (db *DB) seedConfigurations() error {
	// 检查是否已存在配置
	var count int64
	db.Model(&models.Configuration{}).Count(&count)
	if count > 0 {
		return nil // 已存在配置，跳过初始化
	}

	// 默认配置项
	configs := []models.Configuration{
		{
			Key:         "scanner.timeout_ms",
			Value:       "100",
			Description: "扫码枪输入超时时间（毫秒）",
			Type:        "int",
			Category:    "scanner",
			IsSystem:    true,
		},
		{
			Key:         "scanner.min_length",
			Value:       "3",
			Description: "最小条码长度",
			Type:        "int",
			Category:    "scanner",
			IsSystem:    true,
		},
		{
			Key:         "scanner.max_length",
			Value:       "50",
			Description: "最大条码长度",
			Type:        "int",
			Category:    "scanner",
			IsSystem:    true,
		},
		{
			Key:         "websocket.max_connections",
			Value:       "100",
			Description: "WebSocket最大连接数",
			Type:        "int",
			Category:    "websocket",
			IsSystem:    true,
		},
		{
			Key:         "system.auto_cleanup_days",
			Value:       "30",
			Description: "自动清理扫码记录的天数",
			Type:        "int",
			Category:    "system",
			IsSystem:    true,
		},
	}

	return db.CreateInBatches(configs, 10).Error
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// getLogLevel 获取GORM日志级别
func getLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Info
	}
}

// Health 健康检查
func (db *DB) Health() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// GetStats 获取数据库统计信息
func (db *DB) GetStats() map[string]interface{} {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
