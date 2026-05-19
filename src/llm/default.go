package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// DefaultCompletion 调用默认 OpenAI 兼容 Chat Completions API 处理纯文本输入。
func DefaultCompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	return defaultChatCompletion(systemPrompt, userContent, nil, options)
}

// DefaultResponse 调用默认 OpenAI 兼容 Chat Completions API 处理图文输入。
func DefaultResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	return defaultChatCompletion(systemPrompt, userContent, imageURLs, options)
}

func defaultChatCompletion(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	client := newOpenAIClient(options)
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	if len(imageURLs) == 0 {
		messages = append(messages, openai.UserMessage(userContent))
	} else {
		messages = append(messages, openai.UserMessage(defaultContentParts(userContent, imageURLs)))
	}

	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    shared.ChatModel(options.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("调用默认 Chat Completions API 失败: %w", err)
	}

	return openAICompletionToChatCompletion(resp), nil
}

func defaultContentParts(userContent string, imageURLs []string) []openai.ChatCompletionContentPartUnionParam {
	parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(imageURLs)+1)
	if userContent != "" {
		parts = append(parts, openai.TextContentPart(userContent))
	}
	for _, imageURL := range imageURLs {
		if imageURL == "" {
			continue
		}
		parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL: imageURL,
		}))
	}
	return parts
}
