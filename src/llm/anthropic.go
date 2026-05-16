package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicDefaultMaxTokens int64 = 4096

// AnthropicCompletion 调用 Anthropic Messages API 处理纯文本输入。
func AnthropicCompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	return anthropicMessage(systemPrompt, userContent, nil, options)
}

// AnthropicResponse 调用 Anthropic Messages API 处理图文输入。
func AnthropicResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	return anthropicMessage(systemPrompt, userContent, imageURLs, options)
}

// anthropicMessage 构造 Anthropic 消息请求并返回统一响应。
func anthropicMessage(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	client := anthropic.NewClient(anthropicClientOptions(options)...)
	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		MaxTokens: anthropicDefaultMaxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropicContentBlocks(userContent, imageURLs)...),
		},
		Model:  anthropic.Model(options.Model),
		System: anthropicSystemPrompt(systemPrompt),
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Anthropic Messages API 失败: %w", err)
	}

	return anthropicMessageToChatCompletion(resp), nil
}

// anthropicClientOptions 根据配置生成 Anthropic 客户端选项。
func anthropicClientOptions(options LLMOptions) []option.RequestOption {
	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
	}
	if options.BaseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(options.BaseURL))
	}
	return clientOptions
}

// anthropicSystemPrompt 将系统提示词转换为 Anthropic system 参数。
func anthropicSystemPrompt(systemPrompt string) []anthropic.TextBlockParam {
	if systemPrompt == "" {
		return nil
	}
	return []anthropic.TextBlockParam{
		{Text: systemPrompt},
	}
}

// anthropicContentBlocks 将用户文本和图片转换为 Anthropic 内容块。
func anthropicContentBlocks(userContent string, imageURLs []string) []anthropic.ContentBlockParamUnion {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(imageURLs)+1)
	for _, imageURL := range imageURLs {
		if imageURL == "" {
			continue
		}
		block, err := anthropicImageBlock(imageURL)
		if err != nil {
			blocks = append(blocks, anthropic.NewTextBlock(fmt.Sprintf("[图片读取失败: %v]", err)))
			continue
		}
		blocks = append(blocks, block)
	}
	if userContent != "" {
		blocks = append(blocks, anthropic.NewTextBlock(userContent))
	}
	return blocks
}

// anthropicImageBlock 将图片地址、本地路径或 data URL 转换为 Anthropic 图片块。
func anthropicImageBlock(imageURL string) (anthropic.ContentBlockParamUnion, error) {
	if strings.HasPrefix(imageURL, "data:image/") {
		mediaType, data, err := anthropicParseDataURL(imageURL)
		if err != nil {
			return anthropic.ContentBlockParamUnion{}, err
		}
		return anthropic.NewImageBlockBase64(mediaType, data), nil
	}

	if parsedURL, err := strings.CutPrefix(imageURL, "http://"); err && parsedURL != "" {
		return anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: imageURL}), nil
	}
	if parsedURL, err := strings.CutPrefix(imageURL, "https://"); err && parsedURL != "" {
		return anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: imageURL}), nil
	}

	data, err := os.ReadFile(imageURL)
	if err != nil {
		return anthropic.ContentBlockParamUnion{}, err
	}

	mediaType := kimiImageContentType(imageURL)
	if mediaType == "" {
		mediaType = "image/png"
	}
	return anthropic.NewImageBlockBase64(mediaType, base64.StdEncoding.EncodeToString(data)), nil
}

// anthropicParseDataURL 解析图片 data URL 中的媒体类型和 base64 数据。
func anthropicParseDataURL(dataURL string) (string, string, error) {
	header, data, ok := strings.Cut(dataURL, ",")
	if !ok {
		return "", "", fmt.Errorf("无效的 data URL")
	}

	mediaType := strings.TrimPrefix(header, "data:")
	mediaType = strings.TrimSuffix(mediaType, ";base64")
	if !strings.HasPrefix(mediaType, "image/") {
		return "", "", fmt.Errorf("不支持的图片类型: %s", mediaType)
	}
	return mediaType, data, nil
}

// anthropicMessageToChatCompletion 将 Anthropic 响应转换为统一结构。
func anthropicMessageToChatCompletion(resp *anthropic.Message) *ChatCompletionResponse {
	if resp == nil {
		return &ChatCompletionResponse{}
	}

	var content strings.Builder
	var reasoning strings.Builder
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			content.WriteString(block.Text)
		case "thinking":
			reasoning.WriteString(block.Thinking)
		}
	}

	return &ChatCompletionResponse{
		Choices: []ChatCompletionChoice{
			{
				Message: ChatCompletionMessage{
					Role:             "assistant",
					Content:          strings.TrimSpace(content.String()),
					ReasoningContent: strings.TrimSpace(reasoning.String()),
				},
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}
