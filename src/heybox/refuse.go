package heybox

import (
	"fmt"
	"heybox-bot/config"
	"heybox-bot/logger"
	"slices"
	"strconv"
	"time"
)

const refuseFrequencyTimerName = "refuseFrequency"

// initRefuseTimer 初始化拒绝服务相关定时器。
func initRefuseTimer() error {
	if config.GetBotMode() != config.BotModeFrequency {
		return nil
	}
	return AddTimer(refuseFrequencyTimerName, time.Minute, decreaseUserCallCountsCb)
}

func decreaseUserCallCountsCb(_ *TimerContext) error {
	if err := cleanupUserCallTimes(); err != nil {
		return err
	}
	logger.Debug("已清理过期用户调用频率记录")
	return nil
}

// shouldRefuseService 判断指定用户是否不在服务白名单中。
func shouldRefuseService(userID string) (bool, error) {
	if config.GetBotMode() == config.BotModeFrequency {
		count, err := recordUserCall(userID)
		if err != nil {
			return false, err
		}
		return count > config.GetBotFrequency(), nil
	}

	if config.GetBotMode() != config.BotModeWhiteList {
		return false, nil
	}

	parsedUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false, fmt.Errorf("解析用户 ID %q 失败: %w", userID, err)
	}

	if slices.Contains(config.GetBotWhiteList(), parsedUserID) {
		return false, nil
	}
	return true, nil
}
