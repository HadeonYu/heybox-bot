// package config handle config file, provide get config value functions.
// to add new config: set default value or ensure be set in function init(), and write get function
package config

import (
	"fmt"
	"heybox-bot/logger"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	BotModeWhiteList = "white_list"
	BotModeFrequency = "frequency"
)

type runtimeConfig struct {
	LogPath   string
	LogLevel  string
	LogMaxDay int

	BotMode               string
	BotInitWaitTime       time.Duration
	BotMaxWaitTime        time.Duration
	BotMaxPostImageNum    int
	BotMaxCommentImageNum int
	BotWhiteList          []int64
	BotFrequency          int

	LLMBaseURL       string
	LLMVendor        string
	LLMModel         string
	LLMAPIKey        string
	LLMSupportImage  bool
	LLMExtraImageLLM bool

	ImageLLMBaseURL string
	ImageLLMVendor  string
	ImageLLMModel   string
	ImageLLMAPIKey  string
}

var configSnapshot atomic.Value

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

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	cfg, err := loadRuntimeConfig()
	if err != nil {
		return err
	}
	storeRuntimeConfig(cfg)
	logBotMode(cfg)

	viper.OnConfigChange(func(event fsnotify.Event) {
		cfg, err := loadRuntimeConfig()
		if err != nil {
			logger.Error("配置热更新失败，继续使用上一版配置: %v", err)
			return
		}
		storeRuntimeConfig(cfg)
		logger.Info("配置热更新成功: %s", event.Name)
		logBotMode(cfg)
	})
	viper.WatchConfig()
	return nil
}

func setDefaults() {
	viper.SetDefault("log.path", "log/")
	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.max_day", 7)

	viper.SetDefault("bot.init_wait_time", 10)
	viper.SetDefault("bot.max_wait_time", 120)
	viper.SetDefault("bot.max_post_image_num", 3)
	viper.SetDefault("bot.max_comment_image_num", 3)
	viper.SetDefault("bot.white_list", []int{})
	viper.SetDefault("bot.frequency", 3)
	viper.SetDefault("llm.support_image", true)
	viper.SetDefault("llm.extra_image_llm", false)
	viper.SetDefault("llm.chat.vendor", "openai")
	viper.SetDefault("llm.image.vendor", "openai")
}

func loadRuntimeConfig() (*runtimeConfig, error) {
	cfg := &runtimeConfig{
		LogPath:   viper.GetString("log.path"),
		LogLevel:  viper.GetString("log.level"),
		LogMaxDay: viper.GetInt("log.max_day"),

		BotMode:               strings.ToLower(viper.GetString("bot.mode")),
		BotInitWaitTime:       time.Duration(viper.GetInt("bot.init_wait_time")) * time.Second,
		BotMaxWaitTime:        time.Duration(viper.GetInt("bot.max_wait_time")) * time.Second,
		BotMaxPostImageNum:    viper.GetInt("bot.max_post_image_num"),
		BotMaxCommentImageNum: viper.GetInt("bot.max_comment_image_num"),
		BotWhiteList:          intSliceToInt64(viper.GetIntSlice("bot.white_list")),
		BotFrequency:          viper.GetInt("bot.frequency"),

		LLMBaseURL:       viper.GetString("llm.chat.base_url"),
		LLMVendor:        strings.ToLower(viper.GetString("llm.chat.vendor")),
		LLMModel:         viper.GetString("llm.chat.model"),
		LLMAPIKey:        viper.GetString("llm.chat.api_key"),
		LLMSupportImage:  viper.GetBool("llm.support_image"),
		LLMExtraImageLLM: viper.GetBool("llm.extra_image_llm"),

		ImageLLMBaseURL: viper.GetString("llm.image.base_url"),
		ImageLLMVendor:  strings.ToLower(viper.GetString("llm.image.vendor")),
		ImageLLMModel:   viper.GetString("llm.image.model"),
		ImageLLMAPIKey:  viper.GetString("llm.image.api_key"),
	}
	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateRuntimeConfig(cfg *runtimeConfig) error {
	if cfg.BotInitWaitTime <= 0 {
		return fmt.Errorf("配置文件中 bot.init_wait_time 必须大于 0")
	}
	if cfg.BotMaxWaitTime <= 0 {
		return fmt.Errorf("配置文件中 bot.max_wait_time 必须大于 0")
	}
	if cfg.BotMaxWaitTime < cfg.BotInitWaitTime {
		return fmt.Errorf("配置文件中 bot.max_wait_time 不能小于 bot.init_wait_time")
	}
	if cfg.BotMaxPostImageNum < 0 {
		return fmt.Errorf("配置文件中 bot.max_post_image_num 不能小于 0")
	}
	if cfg.BotMaxCommentImageNum < 0 {
		return fmt.Errorf("配置文件中 bot.max_comment_image_num 不能小于 0")
	}
	switch cfg.BotMode {
	case BotModeWhiteList:
		if len(cfg.BotWhiteList) == 0 {
			return fmt.Errorf("配置文件中 bot.mode 为 %q 时 bot.white_list 不能为空", BotModeWhiteList)
		}
	case BotModeFrequency:
		if cfg.BotFrequency <= 0 {
			return fmt.Errorf("配置文件中 bot.mode 为 %q 时 bot.frequency 必须大于 0", BotModeFrequency)
		}
	default:
		return fmt.Errorf("配置文件中 bot.mode 必须为 %q 或 %q", BotModeWhiteList, BotModeFrequency)
	}

	if cfg.LLMBaseURL == "" {
		return fmt.Errorf("配置文件中 llm.chat.base_url 为空")
	}
	if cfg.LLMModel == "" {
		return fmt.Errorf("配置文件中 llm.chat.model 为空")
	}
	if cfg.LLMAPIKey == "" {
		return fmt.Errorf("配置文件或环境变量中 llm.chat.api_key 为空")
	}

	if cfg.LLMExtraImageLLM {
		if cfg.ImageLLMBaseURL == "" {
			return fmt.Errorf("配置文件中 llm.extra_image_llm 为 true 但是 llm.image.base_url 为空")
		}
		if cfg.ImageLLMModel == "" {
			return fmt.Errorf("配置文件中 llm.extra_image_llm 为 true 但是 llm.image.model 为空")
		}
		if cfg.ImageLLMAPIKey == "" {
			return fmt.Errorf("配置文件或环境变量中 llm.extra_image_llm 为 true 但是 llm.image.api_key 为空")
		}
	}
	return nil
}

func storeRuntimeConfig(cfg *runtimeConfig) {
	copyCfg := *cfg
	copyCfg.BotWhiteList = append([]int64(nil), cfg.BotWhiteList...)
	configSnapshot.Store(&copyCfg)
}

func getRuntimeConfig() *runtimeConfig {
	if cfg, ok := configSnapshot.Load().(*runtimeConfig); ok && cfg != nil {
		return cfg
	}
	return &runtimeConfig{}
}

func intSliceToInt64(values []int) []int64 {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		result = append(result, int64(value))
	}
	return result
}

func logBotMode(cfg *runtimeConfig) {
	switch cfg.BotMode {
	case BotModeWhiteList:
		logger.Info("bot 以白名单模式运行")
	case BotModeFrequency:
		logger.Info("bot 以频率限制模式运行")
	}
}

// ------ log --------
func GetLogPath() string {
	return getRuntimeConfig().LogPath
}
func GetLogLevel() string {
	return getRuntimeConfig().LogLevel
}
func GetLogMaxDay() int {
	return getRuntimeConfig().LogMaxDay
}

// ------ bot --------
func GetBotMode() string {
	return getRuntimeConfig().BotMode
}

func GetBotInitWaitTime() time.Duration {
	return getRuntimeConfig().BotInitWaitTime
}

func GetBotMaxWaitTime() time.Duration {
	return getRuntimeConfig().BotMaxWaitTime
}

func GetBotMaxPostImageNum() int {
	return getRuntimeConfig().BotMaxPostImageNum
}

func GetBotMaxCommentImageNum() int {
	return getRuntimeConfig().BotMaxCommentImageNum
}

func GetBotWhiteList() []int64 {
	return append([]int64(nil), getRuntimeConfig().BotWhiteList...)
}

func GetBotFrequency() int {
	return getRuntimeConfig().BotFrequency
}

// ------ llm --------
func GetLLMBaseUrl() string {
	return getRuntimeConfig().LLMBaseURL
}
func GetLLMVendor() string {
	return getRuntimeConfig().LLMVendor
}
func GetLLMModel() string {
	return getRuntimeConfig().LLMModel
}
func GetLLMApiKey() string {
	return getRuntimeConfig().LLMAPIKey
}
func GetLLMSupportImage() bool {
	return getRuntimeConfig().LLMSupportImage
}
func GetLLMExtraImageLLM() bool {
	return getRuntimeConfig().LLMExtraImageLLM
}
func GetImageLLMBaseUrl() string {
	return getRuntimeConfig().ImageLLMBaseURL
}
func GetImageLLMVendor() string {
	return getRuntimeConfig().ImageLLMVendor
}
func GetImageLLMModel() string {
	return getRuntimeConfig().ImageLLMModel
}
func GetImageLLMApiKey() string {
	return getRuntimeConfig().ImageLLMAPIKey
}
