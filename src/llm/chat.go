package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"heybox-bot/config"
	"heybox-bot/logger"
	"io"
	"net/http"
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

type chatCompletionRequest struct {
	Messages       []ChatCompletionMessage `json:"messages"`
	Model          string                  `json:"model"`
	Thinking       map[string]string       `json:"thinking,omitempty"`
	MaxTokens      int                     `json:"max_tokens"`
	ResponseFormat map[string]string       `json:"response_format,omitempty"`
}

func Chat(content string) (*ChatCompletionResponse, error) {
	payload := chatCompletionRequest{
		Messages: []ChatCompletionMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		Model:          config.GetLLMModel(),
		Thinking:       map[string]string{"type": config.GetLLMThinking()},
		MaxTokens:      config.GetLLMMaxTokens(),
		ResponseFormat: map[string]string{"type": "text"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, config.GetLLMBaseUrl(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+config.GetLLMApiKey())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		logger.Error("请求失败，响应内容: %s", string(respBody))
		return nil, fmt.Errorf("请求失败: %s", res.Status)
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		logger.Error("解析响应失败响应内容: %s", string(respBody))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &resp, nil
}

var systemPrompt = `你是一个社区机器人，在社区里你必须发言简短，因为没人喜欢长篇大论。你必须遵守国家法律，坚守道德底线。你可以玩梗，形象是风趣幽默`
