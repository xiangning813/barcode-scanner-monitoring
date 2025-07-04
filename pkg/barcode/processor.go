package barcode

import (
	"strings"
	"time"
)

// BarcodeData 条码数据结构
type BarcodeData struct {
	Content   string    `json:"content"`
	Length    int       `json:"length"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// Processor 条码处理器
type Processor struct{}

// NewProcessor 创建新的条码处理器
func NewProcessor() *Processor {
	return &Processor{}
}

// ProcessBarcode 处理条码数据
func (p *Processor) ProcessBarcode(content string) *BarcodeData {
	timestamp := time.Now()
	
	barcodeData := &BarcodeData{
		Content:   content,
		Length:    len(content),
		Type:      p.GetBarcodeType(content),
		Timestamp: timestamp,
		Status:    "success",
	}
	
	// 业务逻辑处理
	barcodeData.Message = p.generateMessage(content)
	
	return barcodeData
}

// GetBarcodeType 获取条码类型
func (p *Processor) GetBarcodeType(barcode string) string {
	if barcode == "" {
		return "未知"
	}
	
	switch {
	case len(barcode) == 8 && p.isAllDigits(barcode):
		return "EAN-8"
	case len(barcode) == 12 && p.isAllDigits(barcode):
		return "UPC-A"
	case len(barcode) == 13 && p.isAllDigits(barcode):
		return "EAN-13"
	case len(barcode) == 14 && p.isAllDigits(barcode):
		return "ITF-14"
	case p.isAlphaNumeric(barcode):
		return "Code 128"
	case strings.HasPrefix(barcode, "PRD"):
		return "产品条码"
	case strings.HasPrefix(barcode, "LOT"):
		return "批次条码"
	case strings.HasPrefix(barcode, "SN"):
		return "序列号条码"
	default:
		return "其他类型"
	}
}

// generateMessage 生成处理消息
func (p *Processor) generateMessage(barcode string) string {
	switch {
	case strings.HasPrefix(barcode, "PRD"):
		return "识别为产品条码，正在查询产品信息..."
	case strings.HasPrefix(barcode, "LOT"):
		return "识别为批次条码，正在查询批次信息..."
	case strings.HasPrefix(barcode, "SN"):
		return "识别为序列号条码，正在验证序列号..."
	case len(barcode) == 13 && p.isAllDigits(barcode):
		return "识别为EAN-13条码，正在验证..."
	case len(barcode) == 12 && p.isAllDigits(barcode):
		return "识别为UPC-A条码，正在处理..."
	case len(barcode) == 8 && p.isAllDigits(barcode):
		return "识别为EAN-8条码，正在处理..."
	case len(barcode) == 14 && p.isAllDigits(barcode):
		return "识别为ITF-14条码，正在处理..."
	default:
		return "通用条码，正在记录..."
	}
}

// isAllDigits 检查是否全为数字
func (p *Processor) isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isAlphaNumeric 检查是否为字母数字字符
func (p *Processor) isAlphaNumeric(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// ValidateBarcode 验证条码格式
func (p *Processor) ValidateBarcode(barcode string) (bool, string) {
	if barcode == "" {
		return false, "条码不能为空"
	}
	
	if len(barcode) < 3 {
		return false, "条码长度太短"
	}
	
	if len(barcode) > 50 {
		return false, "条码长度太长"
	}
	
	// 检查是否包含非法字符
	for _, r := range barcode {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || 
			r == '-' || r == '.' || r == '_' || r == '/' || r == '\\' || r == ':' || r == ';' || 
			r == '[' || r == ']' || r == '(' || r == ')' || r == '+' || r == '=' || r == ' ') {
			return false, "条码包含非法字符"
		}
	}
	
	return true, "条码格式有效"
}

// GetBarcodeInfo 获取条码详细信息
func (p *Processor) GetBarcodeInfo(barcode string) map[string]interface{} {
	info := map[string]interface{}{
		"content":    barcode,
		"length":     len(barcode),
		"type":       p.GetBarcodeType(barcode),
		"is_numeric": p.isAllDigits(barcode),
		"is_alpha":   p.isAlphaNumeric(barcode),
	}
	
	// 添加特定类型的信息
	switch p.GetBarcodeType(barcode) {
	case "EAN-13":
		info["country_code"] = p.getEAN13CountryCode(barcode)
	case "UPC-A":
		info["manufacturer_code"] = p.getUPCAManufacturerCode(barcode)
	case "产品条码":
		info["product_id"] = strings.TrimPrefix(barcode, "PRD")
	case "批次条码":
		info["lot_number"] = strings.TrimPrefix(barcode, "LOT")
	case "序列号条码":
		info["serial_number"] = strings.TrimPrefix(barcode, "SN")
	}
	
	return info
}

// getEAN13CountryCode 获取EAN-13国家代码
func (p *Processor) getEAN13CountryCode(barcode string) string {
	if len(barcode) != 13 || !p.isAllDigits(barcode) {
		return "未知"
	}
	
	countryCode := barcode[:3]
	switch {
	case countryCode >= "690" && countryCode <= "699":
		return "中国"
	case countryCode >= "000" && countryCode <= "019":
		return "美国/加拿大"
	case countryCode >= "020" && countryCode <= "029":
		return "店内使用"
	case countryCode >= "030" && countryCode <= "039":
		return "美国药品"
	case countryCode >= "400" && countryCode <= "440":
		return "德国"
	case countryCode >= "450" && countryCode <= "459":
		return "日本"
	case countryCode >= "460" && countryCode <= "469":
		return "俄罗斯"
	case countryCode >= "471":
		return "台湾"
	case countryCode >= "480" && countryCode <= "489":
		return "菲律宾"
	default:
		return "其他国家"
	}
}

// getUPCAManufacturerCode 获取UPC-A制造商代码
func (p *Processor) getUPCAManufacturerCode(barcode string) string {
	if len(barcode) != 12 || !p.isAllDigits(barcode) {
		return "未知"
	}
	
	return barcode[:6] // 前6位是制造商代码
}