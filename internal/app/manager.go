package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"userclient/internal/config"
	"userclient/internal/handlers"
	"userclient/internal/routes"
	"userclient/internal/scanner"
	"userclient/internal/websocket"
)

// Manager 应用程序管理器
type Manager struct {
	config          *config.Config
	logger          *logrus.Logger
	hook            *scanner.Hook
	hub             *websocket.Hub
	barcodeHandler  *handlers.BarcodeHandler
	router          *routes.Router
	webSocketServer *http.Server
}

// New 创建应用程序管理器实例
func New() (*Manager, error) {
	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// 初始化WebSocket Hub
	hub := websocket.NewHub(&cfg.WebSocket, logger)

	// 创建条码处理器
	barcodeHandler := handlers.NewBarcodeHandler(hub, logger)

	// 初始化键盘钩子
	hook := scanner.NewHook(&cfg.Scanner, barcodeHandler, logger)

	// 创建路由管理器
	router := routes.New(logger, hub, barcodeHandler)

	return &Manager{
		config:         cfg,
		logger:         logger,
		hook:           hook,
		hub:            hub,
		barcodeHandler: barcodeHandler,
		router:         router,
	}, nil
}

// Start 启动应用程序
func (m *Manager) Start() error {
	m.logger.Info("启动条码扫描器应用程序")

	// 启动WebSocket Hub
	go m.hub.Run()

	// 启动HTTP服务器
	if err := m.startHTTPServer(); err != nil {
		return fmt.Errorf("启动HTTP服务器失败: %w", err)
	}

	// 安装键盘钩子
	if err := m.hook.Install(); err != nil {
		return fmt.Errorf("安装键盘钩子失败: %w", err)
	}

	m.logger.WithField("port", m.config.Server.Port).Info("应用程序启动成功，开始监听设备")

	// 运行消息循环
	m.hook.MessageLoop()
	return nil
}

// Stop 停止应用程序
func (m *Manager) Stop() error {
	m.logger.Info("正在停止应用程序...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 卸载键盘钩子
	if m.hook != nil {
		m.hook.Uninstall()
	}

	// 关闭WebSocket Hub
	if m.hub != nil {
		m.hub.Close()
	}

	// 停止HTTP服务器
	if m.webSocketServer != nil {
		if err := m.webSocketServer.Shutdown(ctx); err != nil {
			m.logger.WithError(err).Error("停止HTTP服务器失败")
		}
	}

	m.logger.Info("应用程序已停止")
	return nil
}

// startHTTPServer 启动HTTP服务器
func (m *Manager) startHTTPServer() error {
	// 设置路由
	engine := m.router.Setup()

	m.webSocketServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.Server.Port),
		Handler: engine,
	}

	go func() {
		m.logger.WithField("port", m.config.Server.Port).Info("启动HTTP服务器")
		if err := m.webSocketServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.WithError(err).Error("HTTP服务器启动失败")
		}
	}()

	return nil
}

// GetLogger 获取日志记录器
func (m *Manager) GetLogger() *logrus.Logger {
	return m.logger
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *config.Config {
	return m.config
}
