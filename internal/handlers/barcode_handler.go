package handlers

import (
	"userclient/internal/websocket"
	"userclient/pkg/barcode"

	"github.com/sirupsen/logrus"
)

// BarcodeHandler 条码处理器
type BarcodeHandler struct {
	hub    *websocket.Hub
	logger *logrus.Logger
}

// NewBarcodeHandler 创建新的条码处理器
func NewBarcodeHandler(hub *websocket.Hub, logger *logrus.Logger) *BarcodeHandler {
	return &BarcodeHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleBarcode 处理条码
func (h *BarcodeHandler) HandleBarcode(content string) error {
	h.logger.WithField("barcode", content).Info("检测到条码")

	// 创建条码处理器来获取详细信息
	processor := barcode.NewProcessor()
	barcodeData := processor.ProcessBarcode(content)

	// 推送到前端
	h.hub.BroadcastBarcode(barcodeData)
	return nil
}
