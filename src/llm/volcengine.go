package llm

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// VolcengineCompletion 调用火山方舟接口处理纯文本输入。
func VolcengineCompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	return volcengineChatCompletion(systemPrompt, userContent, nil, options)
}

// VolcengineResponse 调用火山方舟接口处理图文输入。
func VolcengineResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	return volcengineChatCompletion(systemPrompt, userContent, imageURLs, options)
}

// volcengineChatCompletion 构造火山方舟聊天请求并返回统一响应。
func volcengineChatCompletion(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	clientOptions := make([]arkruntime.ConfigOption, 0, 1)
	if options.BaseURL != "" {
		clientOptions = append(clientOptions, arkruntime.WithBaseUrl(options.BaseURL))
	}

	client := arkruntime.NewClientWithApiKey(options.APIKey, clientOptions...)
	resp, err := client.CreateChatCompletion(context.Background(), arkmodel.CreateChatCompletionRequest{
		Model:    options.Model,
		Messages: volcengineMessages(systemPrompt, userContent, imageURLs),
	})
	if err != nil {
		return nil, fmt.Errorf("调用火山方舟 Chat Completions API 失败: %w", err)
	}

	return volcengineChatCompletionToChatCompletion(resp), nil
}

// volcengineMessages 将系统提示词、用户文本和图片转换为火山消息列表。
func volcengineMessages(systemPrompt, userContent string, imageURLs []string) []*arkmodel.ChatCompletionMessage {
	messages := make([]*arkmodel.ChatCompletionMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, &arkmodel.ChatCompletionMessage{
			Role:    arkmodel.ChatMessageRoleSystem,
			Content: volcengineTextContent(systemPrompt),
		})
	}
	messages = append(messages, &arkmodel.ChatCompletionMessage{
		Role:    arkmodel.ChatMessageRoleUser,
		Content: volcengineUserContent(userContent, imageURLs),
	})
	return messages
}

// volcengineTextContent 将纯文本转换为火山消息内容。
func volcengineTextContent(text string) *arkmodel.ChatCompletionMessageContent {
	return &arkmodel.ChatCompletionMessageContent{
		StringValue: &text,
	}
}

// volcengineUserContent 将用户文本和图片转换为火山用户消息内容。
func volcengineUserContent(userContent string, imageURLs []string) *arkmodel.ChatCompletionMessageContent {
	if len(imageURLs) == 0 {
		return volcengineTextContent(userContent)
	}

	parts := make([]*arkmodel.ChatCompletionMessageContentPart, 0, len(imageURLs)+1)
	if userContent != "" {
		parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
			Type: arkmodel.ChatCompletionMessageContentPartTypeText,
			Text: userContent,
		})
	}
	for _, imageURL := range imageURLs {
		if imageURL == "" {
			continue
		}
		parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
			Type: arkmodel.ChatCompletionMessageContentPartTypeImageURL,
			ImageURL: &arkmodel.ChatMessageImageURL{
				URL:    imageURL,
				Detail: arkmodel.ImageURLDetailAuto,
			},
		})
	}

	return &arkmodel.ChatCompletionMessageContent{
		ListValue: parts,
	}
}

// volcengineChatCompletionToChatCompletion 将火山方舟响应转换为统一结构。
func volcengineChatCompletionToChatCompletion(resp arkmodel.ChatCompletionResponse) *ChatCompletionResponse {
	choices := make([]ChatCompletionChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		if choice == nil {
			continue
		}
		message := ChatCompletionMessage{
			Role: string(choice.Message.Role),
		}
		if choice.Message.Content != nil {
			message.Content = volcengineContentToString(choice.Message.Content)
		}
		if choice.Message.ReasoningContent != nil {
			message.ReasoningContent = *choice.Message.ReasoningContent
		}
		choices = append(choices, ChatCompletionChoice{
			Message: message,
		})
	}

	return &ChatCompletionResponse{
		Choices: choices,
		Usage: ChatCompletionUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			PromptTokensDetails: PromptTokensDetails{
				CachedTokens: resp.Usage.PromptTokensDetails.CachedTokens,
			},
			CompletionTokensDetails: CompletionTokensDetails{
				ReasoningTokens: resp.Usage.CompletionTokensDetails.ReasoningTokens,
			},
		},
	}
}

// volcengineContentToString 从火山消息内容中提取文本。
func volcengineContentToString(content *arkmodel.ChatCompletionMessageContent) string {
	if content == nil {
		return ""
	}
	if content.StringValue != nil {
		return *content.StringValue
	}
	for _, part := range content.ListValue {
		if part != nil && part.Type == arkmodel.ChatCompletionMessageContentPartTypeText {
			return part.Text
		}
	}
	return ""
}
