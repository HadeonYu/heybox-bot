package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// KimiCompletion 调用 Kimi Chat Completions API 处理纯文本输入。
func KimiCompletion(systemPrompt, userContent string, options LLMOptions) (*ChatCompletionResponse, error) {
	client := newOpenAIClient(options)
	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userContent),
		},
		Model: shared.ChatModel(options.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Kimi Chat Completions API 失败: %w", err)
	}

	return openAICompletionToChatCompletion(resp), nil
}

// KimiResponse 调用 Kimi 多模态接口处理图文输入。
func KimiResponse(systemPrompt, userContent string, imageURLs []string, options LLMOptions) (*ChatCompletionResponse, error) {
	contentParts, err := kimiContentParts(userContent, imageURLs)
	if err != nil {
		return nil, err
	}

	client := newOpenAIClient(options)
	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(contentParts),
		},
		Model: shared.ChatModel(options.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("调用 Kimi 多模态 Chat Completions API 失败: %w", err)
	}

	return openAICompletionToChatCompletion(resp), nil
}

// kimiContentParts 将用户文本和图片转换为 Kimi 消息内容片段。
func kimiContentParts(userContent string, imageURLs []string) ([]openai.ChatCompletionContentPartUnionParam, error) {
	parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(imageURLs)+1)
	for _, imageURL := range imageURLs {
		if imageURL == "" {
			continue
		}
		dataURL, err := kimiImageDataURL(imageURL)
		if err != nil {
			return nil, err
		}
		parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL: dataURL,
		}))
	}
	if userContent != "" {
		parts = append(parts, openai.TextContentPart(userContent))
	}
	return parts, nil
}

// kimiImageDataURL 将图片地址或本地路径转换为 Kimi 需要的 data URL。
func kimiImageDataURL(imageURL string) (string, error) {
	if strings.HasPrefix(imageURL, "data:image/") {
		return imageURL, nil
	}

	data, contentType, err := kimiReadImage(imageURL)
	if err != nil {
		return "", fmt.Errorf("读取 Kimi 图片失败: %w", err)
	}
	if contentType == "" {
		contentType = kimiImageContentType(imageURL)
	}
	if contentType == "" {
		contentType = "image/png"
	}

	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data)), nil
}

// kimiReadImage 从远程地址或本地路径读取图片数据和类型。
func kimiReadImage(imageURL string) ([]byte, string, error) {
	parsedURL, err := url.Parse(imageURL)
	if err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
		resp, err := http.Get(imageURL)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, "", fmt.Errorf("下载图片失败: %s", resp.Status)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", err
		}
		return data, kimiNormalizeImageContentType(resp.Header.Get("Content-Type")), nil
	}

	data, err := os.ReadFile(imageURL)
	if err != nil {
		return nil, "", err
	}
	return data, kimiImageContentType(imageURL), nil
}

// kimiImageContentType 根据图片路径或 URL 后缀推断图片媒体类型。
func kimiImageContentType(imageURL string) string {
	parsedURL, err := url.Parse(imageURL)
	imagePath := imageURL
	if err == nil && parsedURL.Path != "" {
		imagePath = parsedURL.Path
	}

	contentType := mime.TypeByExtension(filepath.Ext(imagePath))
	return kimiNormalizeImageContentType(contentType)
}

// kimiNormalizeImageContentType 规范化并校验图片媒体类型。
func kimiNormalizeImageContentType(contentType string) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if strings.HasPrefix(contentType, "image/") {
		return contentType
	}
	return ""
}
