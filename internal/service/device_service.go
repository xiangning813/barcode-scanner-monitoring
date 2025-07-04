package service

import (
	"fmt"
	"time"
	
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	
	"userclient/internal/models"
)

// DeviceService 设备服务
type DeviceService struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewDeviceService 创建设备服务
func NewDeviceService(db *gorm.DB, logger *logrus.Logger) *DeviceService {
	return &DeviceService{
		db:     db,
		logger: logger,
	}
}

// GetDevices 获取设备列表
func (s *DeviceService) GetDevices(page, pageSize int, status string) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var total int64
	
	query := s.db.Model(&models.Device{})
	
	// 添加状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}
	
	return devices, total, nil
}

// GetDevice 获取单个设备
func (s *DeviceService) GetDevice(id uint) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceByName 根据名称获取设备
func (s *DeviceService) GetDeviceByName(name string) (*models.Device, error) {
	var device models.Device
	if err := s.db.Where("name = ?", name).First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// CreateDevice 创建设备
func (s *DeviceService) CreateDevice(device *models.Device) error {
	// 检查设备名称是否已存在
	var existingDevice models.Device
	if err := s.db.Where("name = ?", device.Name).First(&existingDevice).Error; err == nil {
		return fmt.Errorf("设备名称 '%s' 已存在", device.Name)
	}
	
	// 设置默认值
	if device.Status == "" {
		device.Status = "active"
	}
	
	if device.Type == "" {
		device.Type = "scanner"
	}
	
	// 如果是第一个设备，设置为活跃状态
	var count int64
	if err := s.db.Model(&models.Device{}).Count(&count).Error; err == nil && count == 0 {
		device.IsActive = true
	}
	
	if err := s.db.Create(device).Error; err != nil {
		s.logger.WithError(err).Error("创建设备失败")
		return fmt.Errorf("创建设备失败: %w", err)
	}
	
	s.logger.WithField("device_id", device.ID).WithField("device_name", device.Name).Info("设备创建成功")
	return nil
}

// UpdateDevice 更新设备
func (s *DeviceService) UpdateDevice(id uint, updates map[string]interface{}) error {
	// 检查设备是否存在
	var device models.Device
	if err := s.db.First(&device, id).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}
	
	// 如果更新名称，检查是否重复
	if newName, ok := updates["name"]; ok {
		var existingDevice models.Device
		if err := s.db.Where("name = ? AND id != ?", newName, id).First(&existingDevice).Error; err == nil {
			return fmt.Errorf("设备名称 '%s' 已存在", newName)
		}
	}
	
	// 更新最后修改时间
	updates["updated_at"] = time.Now()
	
	if err := s.db.Model(&device).Updates(updates).Error; err != nil {
		s.logger.WithError(err).Error("更新设备失败")
		return fmt.Errorf("更新设备失败: %w", err)
	}
	
	s.logger.WithField("device_id", id).Info("设备更新成功")
	return nil
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(id uint) error {
	// 检查设备是否存在
	var device models.Device
	if err := s.db.First(&device, id).Error; err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}
	
	// 检查是否有关联的条码记录
	var recordCount int64
	if err := s.db.Model(&models.BarcodeRecord{}).Where("device_id = ?", id).Count(&recordCount).Error; err != nil {
		return fmt.Errorf("检查关联记录失败: %w", err)
	}
	
	if recordCount > 0 {
		return fmt.Errorf("无法删除设备，存在 %d 条关联的条码记录", recordCount)
	}
	
	if err := s.db.Delete(&device).Error; err != nil {
		s.logger.WithError(err).Error("删除设备失败")
		return fmt.Errorf("删除设备失败: %w", err)
	}
	
	s.logger.WithField("device_id", id).WithField("device_name", device.Name).Info("设备删除成功")
	return nil
}

// ActivateDevice 激活设备
func (s *DeviceService) ActivateDevice(id uint) error {
	// 先将所有设备设置为非活跃状态
	if err := s.db.Model(&models.Device{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("取消其他设备激活状态失败: %w", err)
	}
	
	// 激活指定设备
	result := s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":  true,
		"status":     "active",
		"updated_at": time.Now(),
	})
	
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("激活设备失败")
		return fmt.Errorf("激活设备失败: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备不存在")
	}
	
	s.logger.WithField("device_id", id).Info("设备激活成功")
	return nil
}

// DeactivateDevice 停用设备
func (s *DeviceService) DeactivateDevice(id uint) error {
	result := s.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":  false,
		"status":     "inactive",
		"updated_at": time.Now(),
	})
	
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("停用设备失败")
		return fmt.Errorf("停用设备失败: %w", result.Error)
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备不存在")
	}
	
	s.logger.WithField("device_id", id).Info("设备停用成功")
	return nil
}

// GetActiveDevice 获取当前活跃设备
func (s *DeviceService) GetActiveDevice() (*models.Device, error) {
	var device models.Device
	if err := s.db.Where("is_active = ? AND status = ?", true, "active").First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// UpdateDeviceLastSeen 更新设备最后活跃时间
func (s *DeviceService) UpdateDeviceLastSeen(id uint) error {
	return s.db.Model(&models.Device{}).Where("id = ?", id).Update("last_seen_at", time.Now()).Error
}

// GetDeviceStats 获取设备统计信息
func (s *DeviceService) GetDeviceStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 总设备数
	var totalCount int64
	if err := s.db.Model(&models.Device{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount
	
	// 活跃设备数
	var activeCount int64
	if err := s.db.Model(&models.Device{}).Where("status = ?", "active").Count(&activeCount).Error; err != nil {
		return nil, err
	}
	stats["active_count"] = activeCount
	
	// 在线设备数（最近5分钟有活动）
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	var onlineCount int64
	if err := s.db.Model(&models.Device{}).Where("last_seen_at > ?", fiveMinutesAgo).Count(&onlineCount).Error; err != nil {
		return nil, err
	}
	stats["online_count"] = onlineCount
	
	// 按类型统计
	var typeStats []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	if err := s.db.Model(&models.Device{}).Select("type, count(*) as count").Group("type").Find(&typeStats).Error; err != nil {
		return nil, err
	}
	stats["type_stats"] = typeStats
	
	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := s.db.Model(&models.Device{}).Select("status, count(*) as count").Group("status").Find(&statusStats).Error; err != nil {
		return nil, err
	}
	stats["status_stats"] = statusStats
	
	return stats, nil
}

// SearchDevices 搜索设备
func (s *DeviceService) SearchDevices(keyword string, page, pageSize int) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var total int64
	
	query := s.db.Model(&models.Device{})
	
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("name LIKE ? OR type LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}
	
	return devices, total, nil
}

// CleanupInactiveDevices 清理长时间未活跃的设备
func (s *DeviceService) CleanupInactiveDevices(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	// 只清理非活跃状态且长时间未见的设备
	result := s.db.Where("is_active = ? AND status = ? AND (last_seen_at < ? OR last_seen_at IS NULL)", 
		false, "inactive", cutoffDate).Delete(&models.Device{})
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	s.logger.WithField("deleted_count", result.RowsAffected).WithField("cutoff_date", cutoffDate).Info("清理非活跃设备")
	return result.RowsAffected, nil
}