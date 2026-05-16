package llm

import (
	"fmt"
	"heybox-bot/config"
	"strings"
)

const (
	llmVendorOpenAI     = "openai"
	llmVendorDeepSeek   = "deepseek"
	llmVendorVolcengine = "volcengine"
	llmVendorVolcano    = "volcano"
	llmVendorArk        = "ark"
)

type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage    `json:"usage"`
}

type ChatCompletionChoice struct {
	Message ChatCompletionMessage `json:"message"`
}

type ChatCompletionMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content"`
}

type ChatCompletionUsage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
	PromptCacheHitTokens    int                     `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens   int                     `json:"prompt_cache_miss_tokens"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

func GenerateResponse(content string, imageURLs []string) (*ChatCompletionResponse, error) {
	if !config.GetLLMSupportImage() {
		imageURLs = nil
	}

	if len(imageURLs) == 0 || !config.GetLLMExtraImageLLM() {
		return chat(content, imageURLs)
	}

	imageDescription, err := describeImages(imageURLs)
	if err != nil {
		return nil, err
	}

	return chat(joinContentAndImageDescription(content, imageDescription), nil)
}

func chat(content string, imageURLs []string) (*ChatCompletionResponse, error) {
	options := LLMOptions{
		Vendor:  config.GetLLMVendor(),
		BaseURL: config.GetLLMBaseUrl(),
		APIKey:  config.GetLLMApiKey(),
		Model:   config.GetLLMModel(),
	}

	if len(imageURLs) == 0 {
		return callChatLLM(systemPrompt, content, options)
	}
	return callResponseLLM(systemPrompt, content, imageURLs, options)
}

func describeImages(imageURLs []string) (string, error) {
	resp, err := callResponseLLM(imageDescriptionSystemPrompt, imageDescriptionUserPrompt, imageURLs, LLMOptions{
		Vendor:  config.GetImageLLMVendor(),
		BaseURL: config.GetImageLLMBaseUrl(),
		APIKey:  config.GetImageLLMApiKey(),
		Model:   config.GetImageLLMModel(),
	})
	if err != nil {
		return "", fmt.Errorf("生成图片描述失败: %w", err)
	}

	content := firstResponseContent(resp)
	if content == "" {
		return "", fmt.Errorf("图片描述为空")
	}
	return content, nil
}

type LLMOptions struct {
	Vendor  string
	BaseURL string
	APIKey  string
	Model   string
}

func callChatLLM(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	switch options.Vendor {
	case "", llmVendorOpenAI, llmVendorDeepSeek:
		return OpenAICompletion(systemPrompt, userContent, options)
	case llmVendorVolcengine, llmVendorVolcano, llmVendorArk:
		return VolcengineCompletion(systemPrompt, userContent, options)
	default:
		return OpenAICompletion(systemPrompt, userContent, options)
	}
}

func callResponseLLM(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	switch options.Vendor {
	case "", llmVendorOpenAI, llmVendorDeepSeek:
		return OpenAIResponse(systemPrompt, userContent, imageURLs, options)
	case llmVendorVolcengine, llmVendorVolcano, llmVendorArk:
		return VolcengineResponse(systemPrompt, userContent, imageURLs, options)
	default:
		return OpenAIResponse(systemPrompt, userContent, imageURLs, options)
	}
}

func joinContentAndImageDescription(content, imageDescription string) string {
	if strings.TrimSpace(imageDescription) == "" {
		return content
	}
	if strings.TrimSpace(content) == "" {
		return "图片信息：\n" + imageDescription
	}
	return content + "\n\n图片信息：\n" + imageDescription
}

func firstResponseContent(resp *ChatCompletionResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content)
}

var systemPrompt = `你是一个社区机器人，在社区里你必须发言简短，因为没人喜欢长篇大论。你必须遵守国家法律，坚守道德底线。你可以玩梗，形象是风趣幽默`

var imageDescriptionSystemPrompt = `你负责把图片转换成简短、准确的文字描述。只描述图片中和用户讨论可能相关的信息，不要编造。`

var imageDescriptionUserPrompt = `请描述这些图片的主要内容，保留关键文字、人物、物体、场景和可能影响回复的信息。`
