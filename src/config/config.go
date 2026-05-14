// package config handle config file, provide get config value functions.
// to add new config: set default value or ensure be set in function init(), and write get function
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Load() error {
	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	if err := viper.BindEnv("llm.api_key", "API_KEY"); err != nil {
		return fmt.Errorf("绑定 API_KEY 环境变量失败: %w", err)
	}

	// 设置配置默认值或检查必填配置

	// log
	viper.SetDefault("log.path", "log/")
	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("log.max_day", 7)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	// llm
	viper.SetDefault("llm.max_tokens", 4096)
	viper.SetDefault("llm.thinking", true)
	if viper.GetString("llm.base_url") == "" {
		return fmt.Errorf("配置文件中 llm.base_url 为空")
	}
	if viper.GetString("llm.model") == "" {
		return fmt.Errorf("配置文件中 llm.model 为空")
	}
	if viper.GetString("llm.api_key") == "" {
		return fmt.Errorf("配置文件或环境变量中 llm.api_key 为空")
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

// ------ llm --------
func GetLLMBaseUrl() string {
	return viper.GetString("llm.base_url")
}
func GetLLMModel() string {
	return viper.GetString("llm.model")
}
func GetLLMApiKey() string {
	return viper.GetString("llm.api_key")
}
func GetLLMMaxTokens() int {
	return viper.GetInt("llm.max_tokens")
}
func GetLLMThinking() string {
	if viper.GetBool("llm.thinking") {
		return "enabled"
	}
	return "disabled"
}
