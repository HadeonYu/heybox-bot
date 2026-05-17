package heybox

import (
	"fmt"
	"heybox-bot/config"
	"heybox-bot/logger"
	"heybox-bot/metadata"
	"slices"
	"strconv"
)

const refuseFrequencyTimerName = "refuseFrequency"

func cleanupUserCallTimesCb(_ *TimerContext) error {
	if config.GetBotMode() != config.BotModeFrequency {
		return nil
	}
	if err := metadata.CleanupUserCallTimes(); err != nil {
		return err
	}
	logger.Debug("已清理过期用户调用频率记录")
	return nil
}

// shouldRefuseService 判断指定用户是否不在服务白名单中。
func shouldRefuseService(userID string) (bool, error) {
	if config.GetBotMode() == config.BotModeFrequency {
		count, err := metadata.RecordUserCall(userID)
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
