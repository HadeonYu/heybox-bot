// package config handle config file, provide get config value functions.
// to add new config: set default value or ensure be set in function init(), and write get function
package config

import (
	"fmt"
	"heybox-bot/logger"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	BotModeWhiteList = "white_list"
	BotModeFrequency = "frequency"
)

func Load() error {
	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	if err := viper.BindEnv("llm.chat.api_key", "API_KEY"); err != nil {
		return fmt.Errorf("绑定 API_KEY 环境变量失败: %w", err)
	}
	if err := viper.BindEnv("llm.image.api_key", "IMAGE_API_KEY"); err != nil {
		return fmt.Errorf("绑定 IMAGE_API_KEY 环境变量失败: %w", err)
	}

	// 设置配置默认值或检查必填配置

	// log
	viper.SetDefault("log.path", "log/")
	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.max_day", 7)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	// bot
	viper.SetDefault("bot.init_wait_time", 10)
	viper.SetDefault("bot.max_wait_time", 120)
	viper.SetDefault("bot.max_post_image_num", 3)
	viper.SetDefault("bot.max_comment_image_num", 3)
	viper.SetDefault("bot.white_list", []int{})
	viper.SetDefault("bot.frequency", 3)
	if viper.GetInt("bot.init_wait_time") <= 0 {
		return fmt.Errorf("配置文件中 bot.init_wait_time 必须大于 0")
	}
	if viper.GetInt("bot.max_wait_time") <= 0 {
		return fmt.Errorf("配置文件中 bot.max_wait_time 必须大于 0")
	}
	if viper.GetInt("bot.max_wait_time") < viper.GetInt("bot.init_wait_time") {
		return fmt.Errorf("配置文件中 bot.max_wait_time 不能小于 bot.init_wait_time")
	}
	if viper.GetInt("bot.max_post_image_num") < 0 {
		return fmt.Errorf("配置文件中 bot.max_post_image_num 不能小于 0")
	}
	if viper.GetInt("bot.max_comment_image_num") < 0 {
		return fmt.Errorf("配置文件中 bot.max_comment_image_num 不能小于 0")
	}
	switch GetBotMode() {
	case BotModeWhiteList:
		if len(GetBotWhiteList()) == 0 {
			return fmt.Errorf("配置文件中 bot.mode 为 %q 时 bot.white_list 不能为空", BotModeWhiteList)
		}
		logger.Info("bot 以白名单模式启动")
	case BotModeFrequency:
		if viper.GetInt("bot.frequency") <= 0 {
			return fmt.Errorf("配置文件中 bot.mode 为 %q 时 bot.frequency 必须大于 0", BotModeFrequency)
		}
		logger.Info("bot 以频率限制模式启动")
	default:
		return fmt.Errorf("配置文件中 bot.mode 必须为 %q 或 %q", BotModeWhiteList, BotModeFrequency)
	}

	// llm
	viper.SetDefault("llm.support_image", true)
	viper.SetDefault("llm.extra_image_llm", false)
	viper.SetDefault("llm.chat.vendor", "openai")
	viper.SetDefault("llm.image.vendor", "openai")
	if viper.GetString("llm.chat.base_url") == "" {
		return fmt.Errorf("配置文件中 llm.chat.base_url 为空")
	}
	if viper.GetString("llm.chat.model") == "" {
		return fmt.Errorf("配置文件中 llm.chat.model 为空")
	}
	if viper.GetString("llm.chat.api_key") == "" {
		return fmt.Errorf("配置文件或环境变量中 llm.chat.api_key 为空")
	}

	if viper.GetBool("llm.extra_image_llm") {
		if viper.GetString("llm.image.base_url") == "" {
			return fmt.Errorf("配置文件中 llm.extra_image_llm 为 true 但是 llm.image.base_url 为空")
		}
		if viper.GetString("llm.image.model") == "" {
			return fmt.Errorf("配置文件中 llm.extra_image_llm 为 true 但是 llm.image.model 为空")
		}
		if viper.GetString("llm.image.api_key") == "" {
			return fmt.Errorf("配置文件或环境变量中 llm.extra_image_llm 为 true 但是 llm.image.api_key 为空")
		}
	}

	viper.WatchConfig()
	return nil
}

// ------ log --------
func GetLogPath() string {
	return viper.GetString("log.path")
}
func GetLogLevel() string {
	return viper.GetString("log.level")
}
func GetLogMaxDay() int {
	return viper.GetInt("log.max_day")
}

// ------ bot --------
func GetBotMode() string {
	return strings.ToLower(viper.GetString("bot.mode"))
}

func GetBotInitWaitTime() time.Duration {
	return time.Duration(viper.GetInt("bot.init_wait_time")) * time.Second
}

func GetBotMaxWaitTime() time.Duration {
	return time.Duration(viper.GetInt("bot.max_wait_time")) * time.Second
}

func GetBotMaxPostImageNum() int {
	return viper.GetInt("bot.max_post_image_num")
}

func GetBotMaxCommentImageNum() int {
	return viper.GetInt("bot.max_comment_image_num")
}

func GetBotWhiteList() []int64 {
	whiteList := viper.GetIntSlice("bot.white_list")
	result := make([]int64, 0, len(whiteList))
	for _, userID := range whiteList {
		result = append(result, int64(userID))
	}
	return result
}

func GetBotFrequency() int {
	return viper.GetInt("bot.frequency")
}

// ------ llm --------
func GetLLMBaseUrl() string {
	return viper.GetString("llm.chat.base_url")
}
func GetLLMVendor() string {
	return strings.ToLower(viper.GetString("llm.chat.vendor"))
}
func GetLLMModel() string {
	return viper.GetString("llm.chat.model")
}
func GetLLMApiKey() string {
	return viper.GetString("llm.chat.api_key")
}
func GetLLMSupportImage() bool {
	return viper.GetBool("llm.support_image")
}
func GetLLMExtraImageLLM() bool {
	return viper.GetBool("llm.extra_image_llm")
}
func GetImageLLMBaseUrl() string {
	return viper.GetString("llm.image.base_url")
}
func GetImageLLMVendor() string {
	return strings.ToLower(viper.GetString("llm.image.vendor"))
}
func GetImageLLMModel() string {
	return viper.GetString("llm.image.model")
}
func GetImageLLMApiKey() string {
	return viper.GetString("llm.image.api_key")
}
