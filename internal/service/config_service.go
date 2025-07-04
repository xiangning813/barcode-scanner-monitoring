package service

import (
	"fmt"
	"time"
	
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	
	"userclient/internal/models"
)

// ConfigService 配置服务
type ConfigService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewConfigService 创建配置服务
func NewConfigService(db *gorm.DB, logger *logrus.Logger) *ConfigService {
	return &ConfigService{
		db:     db,
		logger: logger,
	}
}

// GetConfigurations 获取配置列表
func (s *ConfigService) GetConfigurations(category string) ([]*models.Configuration, error) {
	var configs []*models.Configuration
	
	query := s.db.Model(&models.Configuration{})
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	if err := query.Order("category, key").Find(&configs).Error; err != nil {
		return nil, err
	}
	
	return configs, nil
}

// GetConfiguration 获取单个配置
func (s *ConfigService) GetConfiguration(key string) (*models.Configuration, error) {
	var config models.Configuration
	if err := s.db.Where("key = ?", key).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetConfigurationByID 根据ID获取配置
func (s *ConfigService) GetConfigurationByID(id uint) (*models.Configuration, error) {
	var config models.Configuration
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// SetConfiguration 设置配置
func (s *ConfigService) SetConfiguration(key, value, category, description string) error {
	var config models.Configuration
	
	// 查找现有配置
	err := s.db.Where("key = ?", key).First(&config).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询配置失败: %w", err)
	}
	
	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = models.Configuration{
			Key:         key,
			Value:       value,
			Category:    category,
			Description: description,
		}
		
		if err := s.db.Create(&config).Error; err != nil {
			s.logger.WithError(err).Error("创建配置失败")
			return fmt.Errorf("创建配置失败: %w", err)
		}
		
		s.logger.WithField("key", key).WithField("value", value).Info("配置创建成功")
	} else {
		// 更新现有配置
		updates := map[string]interface{}{
			"value":      value,
			"updated_at": time.Now(),
		}
		
		if category != "" {
			updates["category"] = category
		}
		
		if description != "" {
			updates["description"] = description
		}
		
		if err := s.db.Model(&config).Updates(updates).Error; err != nil {
			s.logger.WithError(err).Error("更新配置失败")
			return fmt.Errorf("更新配置失败: %w", err)
		}
		
		s.logger.WithField("key", key).WithField("value", value).Info("配置更新成功")
	}
	
	return nil
}

// UpdateConfiguration 更新配置
func (s *ConfigService) UpdateConfiguration(id uint, updates map[string]interface{}) error {
	// 检查配置是否存在
	var config models.Configuration
	if err := s.db.First(&config, id).Error; err != nil {
		return fmt.Errorf("配置不存在: %w", err)
	}
	
	// 如果更新键名，检查是否重复
	if newKey, ok := updates["key"]; ok {
		var existingConfig models.Configuration
		if err := s.db.Where("key = ? AND id != ?", newKey, id).First(&existingConfig).Error; err == nil {
			return fmt.Errorf("配置键 '%s' 已存在", newKey)
		}
	}
	
	// 更新最后修改时间
	updates["updated_at"] = time.Now()
	
	if err := s.db.Model(&config).Updates(updates).Error; err != nil {
		s.logger.WithError(err).Error("更新配置失败")
		return fmt.Errorf("更新配置失败: %w", err)
	}
	
	s.logger.WithField("config_id", id).WithField("key", config.Key).Info("配置更新成功")
	return nil
}

// DeleteConfiguration 删除配置
func (s *ConfigService) DeleteConfiguration(id uint) error {
	// 检查配置是否存在
	var config models.Configuration
	if err := s.db.First(&config, id).Error; err != nil {
		return fmt.Errorf("配置不存在: %w", err)
	}
	
	if err := s.db.Delete(&config).Error; err != nil {
		s.logger.WithError(err).Error("删除配置失败")
		return fmt.Errorf("删除配置失败: %w", err)
	}
	
	s.logger.WithField("config_id", id).WithField("key", config.Key).Info("配置删除成功")
	return nil
}

// GetConfigurationsByCategory 按分类获取配置
func (s *ConfigService) GetConfigurationsByCategory(category string) (map[string]string, error) {
	var configs []*models.Configuration
	
	if err := s.db.Where("category = ?", category).Find(&configs).Error; err != nil {
		return nil, err
	}
	
	result := make(map[string]string)
	for _, config := range configs {
		result[config.Key] = config.Value
	}
	
	return result, nil
}

// GetAllConfigurations 获取所有配置（按分类分组）
func (s *ConfigService) GetAllConfigurations() (map[string]map[string]string, error) {
	var configs []*models.Configuration
	
	if err := s.db.Order("category, key").Find(&configs).Error; err != nil {
		return nil, err
	}
	
	result := make(map[string]map[string]string)
	for _, config := range configs {
		if result[config.Category] == nil {
			result[config.Category] = make(map[string]string)
		}
		result[config.Category][config.Key] = config.Value
	}
	
	return result, nil
}

// BatchSetConfigurations 批量设置配置
func (s *ConfigService) BatchSetConfigurations(configs []models.Configuration) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, config := range configs {
		var existingConfig models.Configuration
		err := tx.Where("key = ?", config.Key).First(&existingConfig).Error
		
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("查询配置失败: %w", err)
		}
		
		if err == gorm.ErrRecordNotFound {
			// 创建新配置
			if err := tx.Create(&config).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建配置失败: %w", err)
			}
		} else {
			// 更新现有配置
			updates := map[string]interface{}{
				"value":       config.Value,
				"category":    config.Category,
				"description": config.Description,
				"updated_at":  time.Now(),
			}
			
			if err := tx.Model(&existingConfig).Updates(updates).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("更新配置失败: %w", err)
			}
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	s.logger.WithField("count", len(configs)).Info("批量设置配置成功")
	return nil
}

// SearchConfigurations 搜索配置
func (s *ConfigService) SearchConfigurations(keyword string, category string) ([]*models.Configuration, error) {
	var configs []*models.Configuration
	
	query := s.db.Model(&models.Configuration{})
	
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("key LIKE ? OR value LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	if err := query.Order("category, key").Find(&configs).Error; err != nil {
		return nil, err
	}
	
	return configs, nil
}

// GetCategories 获取所有配置分类
func (s *ConfigService) GetCategories() ([]string, error) {
	var categories []string
	
	if err := s.db.Model(&models.Configuration{}).Distinct("category").Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	
	return categories, nil
}

// ExportConfigurations 导出配置
func (s *ConfigService) ExportConfigurations(category string) ([]*models.Configuration, error) {
	var configs []*models.Configuration
	
	query := s.db.Model(&models.Configuration{})
	
	if category != "" {
		query = query.Where("category = ?", category)
	}
	
	if err := query.Order("category, key").Find(&configs).Error; err != nil {
		return nil, err
	}
	
	return configs, nil
}

// ImportConfigurations 导入配置
func (s *ConfigService) ImportConfigurations(configs []*models.Configuration, overwrite bool) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, config := range configs {
		var existingConfig models.Configuration
		err := tx.Where("key = ?", config.Key).First(&existingConfig).Error
		
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("查询配置失败: %w", err)
		}
		
		if err == gorm.ErrRecordNotFound {
			// 创建新配置
			if err := tx.Create(config).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建配置失败: %w", err)
			}
		} else if overwrite {
			// 覆盖现有配置
			updates := map[string]interface{}{
				"value":       config.Value,
				"category":    config.Category,
				"description": config.Description,
				"updated_at":  time.Now(),
			}
			
			if err := tx.Model(&existingConfig).Updates(updates).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("更新配置失败: %w", err)
			}
		}
		// 如果不覆盖且配置已存在，则跳过
	}
	
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	s.logger.WithField("count", len(configs)).WithField("overwrite", overwrite).Info("导入配置成功")
	return nil
}

// ResetConfigurations 重置配置到默认值
func (s *ConfigService) ResetConfigurations(category string) error {
	// 定义默认配置
	defaultConfigs := s.getDefaultConfigurations()
	
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, config := range defaultConfigs {
		if category != "" && config.Category != category {
			continue
		}
		
		var existingConfig models.Configuration
		err := tx.Where("key = ?", config.Key).First(&existingConfig).Error
		
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("查询配置失败: %w", err)
		}
		
		if err == gorm.ErrRecordNotFound {
			// 创建默认配置
			if err := tx.Create(&config).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建默认配置失败: %w", err)
			}
		} else {
			// 重置为默认值
			updates := map[string]interface{}{
				"value":       config.Value,
				"description": config.Description,
				"updated_at":  time.Now(),
			}
			
			if err := tx.Model(&existingConfig).Updates(updates).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("重置配置失败: %w", err)
			}
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	s.logger.WithField("category", category).Info("重置配置成功")
	return nil
}

// getDefaultConfigurations 获取默认配置
func (s *ConfigService) getDefaultConfigurations() []models.Configuration {
	return []models.Configuration{
		{Key: "scanner.timeout", Value: "3000", Category: "scanner", Description: "条码扫描超时时间（毫秒）"},
		{Key: "scanner.min_length", Value: "3", Category: "scanner", Description: "条码最小长度"},
		{Key: "scanner.max_length", Value: "50", Category: "scanner", Description: "条码最大长度"},
		{Key: "scanner.auto_clear", Value: "true", Category: "scanner", Description: "自动清除缓冲区"},
		{Key: "websocket.port", Value: "8080", Category: "websocket", Description: "WebSocket服务端口"},
		{Key: "websocket.max_connections", Value: "100", Category: "websocket", Description: "最大WebSocket连接数"},
		{Key: "api.port", Value: "8081", Category: "api", Description: "HTTP API服务端口"},
		{Key: "api.cors_enabled", Value: "true", Category: "api", Description: "启用CORS"},
		{Key: "database.max_idle_conns", Value: "10", Category: "database", Description: "数据库最大空闲连接数"},
		{Key: "database.max_open_conns", Value: "100", Category: "database", Description: "数据库最大打开连接数"},
		{Key: "log.level", Value: "info", Category: "log", Description: "日志级别"},
		{Key: "log.file_enabled", Value: "true", Category: "log", Description: "启用文件日志"},
		{Key: "security.rate_limit", Value: "100", Category: "security", Description: "API速率限制（每分钟请求数）"},
		{Key: "security.jwt_secret", Value: "your-secret-key", Category: "security", Description: "JWT密钥"},
	}
}