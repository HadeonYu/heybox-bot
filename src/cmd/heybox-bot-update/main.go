package main

import (
	"fmt"
	"heybox-bot/update"
	"os"
	"time"
)

func main() {
	result, err := update.Check()
	if err != nil {
		fmt.Printf("检查更新失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("最新版本: %s\n", result.LatestVersion)
	if result.LegacyVersion {
		fmt.Println("当前版本: 未知（当前 heybox-bot 不支持 --version）")
	} else {
		fmt.Printf("当前版本: %s\n", result.CurrentVersion)
	}

	if result.NeedUpdate {
		fmt.Println("需要更新")
		return
	}

	countdownExit(3, "已是最新版本")
}

func countdownExit(second int, content string) {
	for sec := second; sec > 0; sec-- {
		fmt.Printf("\r%s，%ds 后退出", content, sec)
		time.Sleep(time.Second)
	}
	fmt.Println("")
}
