package models

import (
	"time"
	"gorm.io/gorm"
)

// BarcodeRecord 扫码记录模型
type BarcodeRecord struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Content   string         `json:"content" gorm:"not null;index" validate:"required,min=1,max=100"`
	Length    int            `json:"length" gorm:"not null"`
	Type      string         `json:"type" gorm:"size:50;index"`
	Status    string         `json:"status" gorm:"size:20;default:success"`
	Message   string         `json:"message" gorm:"size:255"`
	DeviceID  *uint          `json:"device_id" gorm:"index"`
	Device    *Device        `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (BarcodeRecord) TableName() string {
	return "barcode_records"
}

// Device 设备模型
type Device struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:100" validate:"required,min=1,max=100"`
	Type        string         `json:"type" gorm:"size:50;default:scanner"`
	Model       string         `json:"model" gorm:"size:100"`
	SerialNo    string         `json:"serial_no" gorm:"size:100;uniqueIndex"`
	Description string         `json:"description" gorm:"size:255"`
	Status      string         `json:"status" gorm:"size:20;default:active"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	LastSeen    *time.Time     `json:"last_seen"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联关系
	BarcodeRecords []BarcodeRecord `json:"barcode_records,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// Configuration 系统配置模型
type Configuration struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Key         string         `json:"key" gorm:"not null;uniqueIndex;size:100" validate:"required"`
	Value       string         `json:"value" gorm:"type:text"`
	Description string         `json:"description" gorm:"size:255"`
	Type        string         `json:"type" gorm:"size:20;default:string"` // string, int, bool, json
	Category    string         `json:"category" gorm:"size:50;index"`
	IsSystem    bool           `json:"is_system" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Configuration) TableName() string {
	return "configurations"
}

// SystemLog 系统日志模型
type SystemLog struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Level     string    `json:"level" gorm:"size:10;index"`
	Message   string    `json:"message" gorm:"type:text"`
	Module    string    `json:"module" gorm:"size:50;index"`
	Action    string    `json:"action" gorm:"size:100"`
	UserID    *uint     `json:"user_id" gorm:"index"`
	IP        string    `json:"ip" gorm:"size:45"`
	UserAgent string    `json:"user_agent" gorm:"size:255"`
	Extra     string    `json:"extra" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (SystemLog) TableName() string {
	return "system_logs"
}