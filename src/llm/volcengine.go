package llm

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func VolcengineCompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	return volcengineChatCompletion(systemPrompt, userContent, nil, options)
}

func VolcengineResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	return volcengineChatCompletion(systemPrompt, userContent, imageURLs, options)
}

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

func volcengineTextContent(text string) *arkmodel.ChatCompletionMessageContent {
	return &arkmodel.ChatCompletionMessageContent{
		StringValue: &text,
	}
}

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
