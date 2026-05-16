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

	llmTestSystemPrompt = "你是一个只能助手"
	llmTestUserContent  = "你好"
	llmTestImagePrompt  = "回答必须以“我看到了这张图片，内容是...”开头"
	llmTestImageContent = "你看到了什么？"
	llmTestImageURL     = "https://ark-project.tos-cn-beijing.volces.com/doc_image/ark_demo_img_1.png"
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

func LLMTest() {
	supportImage := config.GetLLMSupportImage()
	extraImageLLM := config.GetLLMExtraImageLLM()
	fmt.Printf("支持图片: %v\n", supportImage)
	fmt.Printf("额外图像处理大模型: %v\n", extraImageLLM)

	fmt.Println("纯文本测试:")
	textResp, err := callChatLLM(llmTestSystemPrompt, llmTestUserContent, chatOptions())
	if err != nil {
		fmt.Printf("失败: %v\n", err)
	} else {
		fmt.Printf("成功: %s\n", firstResponseContent(textResp))
		printUsage(textResp)
	}

	fmt.Println("图片测试:")
	if !supportImage {
		fmt.Println("跳过: 配置不支持图片")
		return
	}

	var imageResp *ChatCompletionResponse
	if extraImageLLM {
		imageResp, err = callResponseLLM(llmTestImagePrompt, llmTestImageContent, []string{llmTestImageURL}, imageOptions())
	} else {
		imageResp, err = callResponseLLM(llmTestImagePrompt, llmTestImageContent, []string{llmTestImageURL}, chatOptions())
	}
	if err != nil {
		fmt.Printf("失败: %v\n", err)
		return
	}
	fmt.Printf("成功: %s\n", firstResponseContent(imageResp))
	printUsage(imageResp)
}

func chat(content string, imageURLs []string) (*ChatCompletionResponse, error) {
	options := chatOptions()

	if len(imageURLs) == 0 {
		return callChatLLM(systemPrompt, content, options)
	}
	return callResponseLLM(systemPrompt, content, imageURLs, options)
}

func chatOptions() LLMOptions {
	return LLMOptions{
		Vendor:  config.GetLLMVendor(),
		BaseURL: config.GetLLMBaseUrl(),
		APIKey:  config.GetLLMApiKey(),
		Model:   config.GetLLMModel(),
	}
}

func imageOptions() LLMOptions {
	return LLMOptions{
		Vendor:  config.GetImageLLMVendor(),
		BaseURL: config.GetImageLLMBaseUrl(),
		APIKey:  config.GetImageLLMApiKey(),
		Model:   config.GetImageLLMModel(),
	}
}

func describeImages(imageURLs []string) (string, error) {
	resp, err := callResponseLLM(imageDescriptionSystemPrompt, imageDescriptionUserPrompt, imageURLs, imageOptions())
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

func printUsage(resp *ChatCompletionResponse) {
	if resp == nil {
		return
	}
	fmt.Printf("token 消耗: prompt=%d, completion=%d, total=%d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
}

var systemPrompt = `你是一个社区机器人，在社区里你必须发言简短，因为没人喜欢长篇大论。你必须遵守国家法律，坚守道德底线。你可以玩梗，形象是风趣幽默`

var imageDescriptionSystemPrompt = `你负责把图片转换成简短、准确的文字描述。只描述图片中和用户讨论可能相关的信息，不要编造。`

var imageDescriptionUserPrompt = `请描述这些图片的主要内容，保留关键文字、人物、物体、场景和可能影响回复的信息。`
