package main

import (
	"fmt"
	"heybox-bot/update"
	"os"
)

func main() {
	result, err := update.Check()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	fmt.Println("已是最新版本")
}
