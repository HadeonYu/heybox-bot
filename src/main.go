package main

import (
	"heybox-bot/config"
	"heybox-bot/heybox"
	"heybox-bot/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := config.Load(); err != nil {
		logger.OpenDefault()
		defer logger.Close()
		logger.Fatal("加载配置失败: %v", err)
		return
	}

	logger.Open(logger.Options{
		Path:   config.GetLogPath(),
		Level:  config.GetLogLevel(),
		MaxDay: config.GetLogMaxDay(),
	})
	defer logger.Close()

	if err := heybox.Run(); err != nil {
		logger.Error("机器人运行失败: %v", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT)
	defer signal.Stop(sigCh)

	sig := <-sigCh
	logger.Info("收到退出信号: %v", sig)
	heybox.Stop()
}
