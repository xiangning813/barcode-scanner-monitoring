package service

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	
	"userclient/internal/models"
	"userclient/pkg/barcode"
)

// BarcodeService 条码服务
type BarcodeService struct {
	db        *gorm.DB
	processor *barcode.Processor
	logger    *logrus.Logger
}

// NewBarcodeService 创建条码服务
func NewBarcodeService(db *gorm.DB, logger *logrus.Logger) *BarcodeService {
	return &BarcodeService{
		db:        db,
		processor: barcode.NewProcessor(),
		logger:    logger,
	}
}

// HandleBarcode 处理扫描到的条码
func (s *BarcodeService) HandleBarcode(content string) error {
	s.logger.WithField("barcode", content).Info("开始处理条码")
	
	// 验证条码格式
	if valid, msg := s.processor.ValidateBarcode(content); !valid {
		s.logger.WithField("barcode", content).WithField("reason", msg).Warn("条码格式无效")
		return fmt.Errorf("条码格式无效: %s", msg)
	}
	
	// 处理条码数据
	barcodeData := s.processor.ProcessBarcode(content)
	
	// 保存到数据库
	record := &models.BarcodeRecord{
		Content: barcodeData.Content,
		Length:  barcodeData.Length,
		Type:    barcodeData.Type,
		Status:  barcodeData.Status,
		Message: barcodeData.Message,
	}
	
	// 尝试关联设备
	if deviceID := s.getDefaultDeviceID(); deviceID > 0 {
		record.DeviceID = &deviceID
	}
	
	if err := s.db.Create(record).Error; err != nil {
		s.logger.WithError(err).Error("保存条码记录失败")
		return fmt.Errorf("保存条码记录失败: %w", err)
	}
	
	s.logger.WithField("record_id", record.ID).Info("条码记录已保存")
	
	// 执行业务逻辑
	if err := s.executeBusinessLogic(barcodeData); err != nil {
		s.logger.WithError(err).Warn("执行业务逻辑失败")
	}
	
	return nil
}

// GetBarcodeRecords 获取条码记录列表
func (s *BarcodeService) GetBarcodeRecords(page, pageSize int, deviceID *uint, barcodeType string) ([]*models.BarcodeRecord, int64, error) {
	var records []*models.BarcodeRecord
	var total int64
	
	query := s.db.Model(&models.BarcodeRecord{}).Preload("Device")
	
	// 添加过滤条件
	if deviceID != nil {
		query = query.Where("device_id = ?", *deviceID)
	}
	
	if barcodeType != "" {
		query = query.Where("type = ?", barcodeType)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	
	return records, total, nil
}

// GetBarcodeRecord 获取单个条码记录
func (s *BarcodeService) GetBarcodeRecord(id uint) (*models.BarcodeRecord, error) {
	var record models.BarcodeRecord
	if err := s.db.Preload("Device").First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteBarcodeRecord 删除条码记录
func (s *BarcodeService) DeleteBarcodeRecord(id uint) error {
	return s.db.Delete(&models.BarcodeRecord{}, id).Error
}

// GetBarcodeStats 获取条码统计信息
func (s *BarcodeService) GetBarcodeStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 总条码数
	var totalCount int64
	if err := s.db.Model(&models.BarcodeRecord{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount
	
	// 今日条码数
	today := time.Now().Truncate(24 * time.Hour)
	var todayCount int64
	if err := s.db.Model(&models.BarcodeRecord{}).Where("created_at >= ?", today).Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats["today_count"] = todayCount
	
	// 按类型统计
	var typeStats []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	if err := s.db.Model(&models.BarcodeRecord{}).Select("type, count(*) as count").Group("type").Find(&typeStats).Error; err != nil {
		return nil, err
	}
	stats["type_stats"] = typeStats
	
	// 最近7天统计
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	var recentStats []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	if err := s.db.Model(&models.BarcodeRecord{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", sevenDaysAgo).
		Group("DATE(created_at)").
		Order("date").
		Find(&recentStats).Error; err != nil {
		return nil, err
	}
	stats["recent_stats"] = recentStats
	
	return stats, nil
}

// CleanupOldRecords 清理旧记录
func (s *BarcodeService) CleanupOldRecords(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.BarcodeRecord{})
	if result.Error != nil {
		return 0, result.Error
	}
	
	s.logger.WithField("deleted_count", result.RowsAffected).WithField("cutoff_date", cutoffDate).Info("清理旧条码记录")
	return result.RowsAffected, nil
}

// SearchBarcodes 搜索条码
func (s *BarcodeService) SearchBarcodes(keyword string, page, pageSize int) ([]*models.BarcodeRecord, int64, error) {
	var records []*models.BarcodeRecord
	var total int64
	
	query := s.db.Model(&models.BarcodeRecord{}).Preload("Device")
	
	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("content LIKE ? OR type LIKE ? OR message LIKE ?", keyword, keyword, keyword)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	
	return records, total, nil
}

// getDefaultDeviceID 获取默认设备ID
func (s *BarcodeService) getDefaultDeviceID() uint {
	var device models.Device
	if err := s.db.Where("is_active = ? AND status = ?", true, "active").First(&device).Error; err != nil {
		return 0
	}
	return device.ID
}

// executeBusinessLogic 执行业务逻辑
func (s *BarcodeService) executeBusinessLogic(barcodeData *barcode.BarcodeData) error {
	// 根据条码类型执行不同的业务逻辑
	switch {
	case strings.HasPrefix(barcodeData.Content, "PRD"):
		return s.handleProductBarcode(barcodeData)
	case strings.HasPrefix(barcodeData.Content, "LOT"):
		return s.handleLotBarcode(barcodeData)
	case strings.HasPrefix(barcodeData.Content, "SN"):
		return s.handleSerialBarcode(barcodeData)
	case barcodeData.Type == "EAN-13" || barcodeData.Type == "UPC-A":
		return s.handleStandardBarcode(barcodeData)
	default:
		return s.handleGenericBarcode(barcodeData)
	}
}

// handleProductBarcode 处理产品条码
func (s *BarcodeService) handleProductBarcode(barcodeData *barcode.BarcodeData) error {
	s.logger.WithField("barcode", barcodeData.Content).Info("处理产品条码")
	// 这里可以添加产品查询、库存检查等逻辑
	return nil
}

// handleLotBarcode 处理批次条码
func (s *BarcodeService) handleLotBarcode(barcodeData *barcode.BarcodeData) error {
	s.logger.WithField("barcode", barcodeData.Content).Info("处理批次条码")
	// 这里可以添加批次追踪、质量检查等逻辑
	return nil
}

// handleSerialBarcode 处理序列号条码
func (s *BarcodeService) handleSerialBarcode(barcodeData *barcode.BarcodeData) error {
	s.logger.WithField("barcode", barcodeData.Content).Info("处理序列号条码")
	// 这里可以添加序列号验证、设备注册等逻辑
	return nil
}

// handleStandardBarcode 处理标准条码
func (s *BarcodeService) handleStandardBarcode(barcodeData *barcode.BarcodeData) error {
	s.logger.WithField("barcode", barcodeData.Content).Info("处理标准条码")
	// 这里可以添加商品查询、价格检查等逻辑
	return nil
}

// handleGenericBarcode 处理通用条码
func (s *BarcodeService) handleGenericBarcode(barcodeData *barcode.BarcodeData) error {
	s.logger.WithField("barcode", barcodeData.Content).Info("处理通用条码")
	// 这里可以添加通用处理逻辑
	return nil
}