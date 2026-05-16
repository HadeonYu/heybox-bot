package heybox

import (
	"fmt"
	"heybox-bot/config"
	"slices"
	"strconv"
)

// shouldRefuseService 判断指定用户是否不在服务白名单中。
func shouldRefuseService(userID string) (bool, error) {
	parsedUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false, fmt.Errorf("解析用户 ID %q 失败: %w", userID, err)
	}

	if slices.Contains(config.GetBotWhiteList(), parsedUserID) {
		return false, nil
	}
	return true, nil
}
