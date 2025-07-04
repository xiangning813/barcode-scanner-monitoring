package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"userclient/internal/app"
)

func main() {
	// 创建应用程序管理器
	manager, err := app.New()
	if err != nil {
		fmt.Printf("创建应用程序失败: %v\n", err)
		os.Exit(1)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动应用程序
	go func() {
		if err := manager.Start(); err != nil {
			manager.GetLogger().WithError(err).Error("应用程序启动失败")
			os.Exit(1)
		}
	}()

	// 等待退出信号
	sig := <-sigChan
	manager.GetLogger().WithField("signal", sig).Info("收到退出信号")

	// 优雅停止应用程序
	if err := manager.Stop(); err != nil {
		manager.GetLogger().WithError(err).Error("停止应用程序失败")
		os.Exit(1)
	}

	fmt.Println("应用程序已安全退出")
}
