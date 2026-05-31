package main

import (
	"flag"
	"fmt"
	"heybox-bot/config"
	"heybox-bot/db"
	"heybox-bot/heybox"
	"heybox-bot/llm"
	"heybox-bot/logger"
	"heybox-bot/metadata"
	"os"
	"os/signal"
	"syscall"
)

var Version = "dev"

func main() {
	llmTest := flag.Bool("llm_test", false, "测试 LLM 配置")
	version := flag.Bool("version", false, "输出版本号")
	flag.Parse()

	if *version {
		fmt.Println(Version)
		return
	}

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

	if err := metadata.Init(); err != nil {
		logger.Fatal("初始化元数据失败: %v", err)
		return
	}

	if *llmTest {
		llm.LLMTest()
		return
	}

	logVersion()

	if err := db.Open(); err != nil {
		logger.Fatal("打开数据库失败")
	}
	defer db.Close()

	if err := heybox.Run(); err != nil {
		logger.Error("机器人运行失败: %v", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(sigCh)

	sig := <-sigCh
	logger.Info("收到退出信号: %v", sig)
	heybox.Stop()
}

func logVersion() {
	// result, err := update.CheckWithCurrentVersion(Version)
	// if err != nil {
	// 	logger.Warn("检查更新失败: %v", err)
	// 	logger.Info("heybox-bot 版本：%s", Version)
	// 	return
	// }

	// if result.NeedUpdate {
	// 	logger.Info("heybox-bot 版本：%s，发现新版本：%s", Version, result.LatestVersion)
	// 	return
	// }

	logger.Info("heybox-bot 版本：%s", Version)
}
