package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

func OpenAIResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	client := newOpenAIClient(options)
	resp, err := client.Responses.New(context.Background(), responses.ResponseNewParams{
		Instructions: openai.String(systemPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: openAIInput(userContent, imageURLs),
		},
		Model: shared.ResponsesModel(options.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("调用 OpenAI Responses API 失败: %w", err)
	}

	return openAIResponseToChatCompletion(resp), nil
}

func OpenAICompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	client := newOpenAIClient(options)
	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userContent),
		},
		Model: shared.ChatModel(options.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("调用 OpenAI Chat Completions API 失败: %w", err)
	}

	return openAICompletionToChatCompletion(resp), nil
}

func newOpenAIClient(options LLMOptions) openai.Client {
	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
	}
	if options.BaseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(options.BaseURL))
	}

	return openai.NewClient(clientOptions...)
}

func openAIInput(userContent string, imageURLs []string) responses.ResponseInputParam {
	if len(imageURLs) == 0 {
		return responses.ResponseInputParam{
			responses.ResponseInputItemParamOfMessage(userContent, responses.EasyInputMessageRoleUser),
		}
	}

	content := make(responses.ResponseInputMessageContentListParam, 0, len(imageURLs)+1)
	if userContent != "" {
		content = append(content, responses.ResponseInputContentParamOfInputText(userContent))
	}
	for _, imageURL := range imageURLs {
		if imageURL == "" {
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				Detail:   responses.ResponseInputImageDetailAuto,
				ImageURL: openai.String(imageURL),
			},
		})
	}

	return responses.ResponseInputParam{
		responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
	}
}

func openAIResponseToChatCompletion(resp *responses.Response) *ChatCompletionResponse {
	if resp == nil {
		return &ChatCompletionResponse{}
	}

	return &ChatCompletionResponse{
		Choices: []ChatCompletionChoice{
			{
				Message: ChatCompletionMessage{
					Role:    "assistant",
					Content: resp.OutputText(),
				},
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
			PromptTokensDetails: PromptTokensDetails{
				CachedTokens: int(resp.Usage.InputTokensDetails.CachedTokens),
			},
			CompletionTokensDetails: CompletionTokensDetails{
				ReasoningTokens: int(resp.Usage.OutputTokensDetails.ReasoningTokens),
			},
		},
	}
}

func openAICompletionToChatCompletion(resp *openai.ChatCompletion) *ChatCompletionResponse {
	if resp == nil {
		return &ChatCompletionResponse{}
	}

	choices := make([]ChatCompletionChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		choices = append(choices, ChatCompletionChoice{
			Message: ChatCompletionMessage{
				Role:    string(choice.Message.Role),
				Content: choice.Message.Content,
			},
		})
	}

	return &ChatCompletionResponse{
		Choices: choices,
		Usage: ChatCompletionUsage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
			PromptTokensDetails: PromptTokensDetails{
				CachedTokens: int(resp.Usage.PromptTokensDetails.CachedTokens),
			},
			CompletionTokensDetails: CompletionTokensDetails{
				ReasoningTokens: int(resp.Usage.CompletionTokensDetails.ReasoningTokens),
			},
		},
	}
}
