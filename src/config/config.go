// package config handle config file, provide get config value functions.
// to add new config: set default value or ensure be set in function init(), and write get function
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
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
