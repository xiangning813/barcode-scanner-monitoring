package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"userclient/internal/handlers"
	"userclient/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Router 路由管理器
type Router struct {
	engine  *gin.Engine
	logger  *logrus.Logger
	hub     *websocket.Hub
	handler *handlers.BarcodeHandler
}

// New 创建新的路由管理器
func New(logger *logrus.Logger, hub *websocket.Hub, handler *handlers.BarcodeHandler) *Router {
	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)

	return &Router{
		engine:  gin.New(),
		logger:  logger,
		hub:     hub,
		handler: handler,
	}
}

// Setup 设置路由
func (r *Router) Setup() *gin.Engine {
	// 添加中间件
	r.engine.Use(r.loggerMiddleware())
	r.engine.Use(gin.Recovery())

	// 设置路由
	r.setupRoutes()

	return r.engine
}

// setupRoutes 设置具体路由
func (r *Router) setupRoutes() {
	// 根路径 - 提供测试页面
	r.engine.GET("/", r.serveTestPage)

	// WebSocket端点
	r.engine.GET("/ws", r.handleWebSocket)

	// API路由组 - 简单的API，不需要版本控制
	api := r.engine.Group("/api")
	{
		// 健康检查
		api.GET("/health", r.healthCheck)

		// 系统状态
		api.GET("/status", r.getStatus)

		// 条码相关API
		api.GET("/barcodes", r.getBarcodes)      // 获取扫码记录
		api.DELETE("/barcodes", r.clearBarcodes) // 清空扫码记录

		// 统计信息
		api.GET("/stats", r.getStats)
	}
}

// serveTestPage 提供测试页面
func (r *Router) serveTestPage(c *gin.Context) {
	// 获取工作目录
	wd, err := os.Getwd()
	if err != nil {
		r.logger.WithError(err).Error("获取工作目录失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		return
	}

	// 构建HTML文件路径
	htmlPath := filepath.Join(wd, "web", "test-socket.html")

	// 检查文件是否存在
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		r.logger.WithField("path", htmlPath).Error("测试页面文件不存在")
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "测试页面文件不存在",
			"message": "请确保 web/test-socket.html 文件存在",
		})
		return
	}

	// 提供HTML文件
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.File(htmlPath)
}

// handleWebSocket 处理WebSocket连接
func (r *Router) handleWebSocket(c *gin.Context) {
	r.hub.HandleWebSocket(c.Writer, c.Request)
}

// healthCheck 健康检查
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "barcode-scanner",
		"timestamp": gin.H{
			"unix": gin.H{
				"seconds": gin.H{
					"value": "current_time",
				},
			},
		},
	})
}

// getStatus 获取系统状态
func (r *Router) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"websocket": gin.H{
			"connected_clients": r.hub.GetClientCount(),
			"status":            "running",
		},
		"scanner": gin.H{
			"status": "listening",
		},
		"server": gin.H{
			"status": "running",
		},
	})
}

// getBarcodes 获取扫码记录
func (r *Router) getBarcodes(c *gin.Context) {
	// 这里应该从数据库或缓存中获取扫码记录
	// 目前返回示例数据
	c.JSON(http.StatusOK, gin.H{
		"data":    []gin.H{},
		"total":   0,
		"message": "暂无扫码记录",
	})
}

// clearBarcodes 清空扫码记录
func (r *Router) clearBarcodes(c *gin.Context) {
	// 这里应该清空数据库中的扫码记录
	r.logger.Info("清空扫码记录")
	c.JSON(http.StatusOK, gin.H{
		"message": "扫码记录已清空",
	})
}

// getStats 获取统计信息
func (r *Router) getStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total_scans":       0,
		"connected_clients": r.hub.GetClientCount(),
		"uptime":            "0s",
		"last_scan":         nil,
	})
}

// loggerMiddleware 日志中间件
func (r *Router) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求信息
		r.logger.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		}).Info("HTTP请求")

		c.Next()
	}
}
