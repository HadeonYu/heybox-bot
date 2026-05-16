package heybox

import (
	"fmt"
	"heybox-bot/heybox/api"
	"heybox-bot/llm"
	"heybox-bot/logger"
	"strings"
)

// reply 根据整理后的 @ 消息生成回复并发布到对应帖子或评论下。
func reply(results []MessageArrangeResult) error {
	if saved_sess == nil {
		return fmt.Errorf("会话为空")
	}

	for _, result := range results {
		if err := replyOne(result); err != nil {
			return fmt.Errorf("回复消息 %d 失败: %w", result.MessageID, err)
		}
		if err := saveLastAtMessageTime(result.Timestamp); err != nil {
			return fmt.Errorf("更新消息 %d 处理时间失败: %w", result.MessageID, err)
		}
	}
	return nil
}

func replyOne(result MessageArrangeResult) error {
	if result.PostLink == nil {
		return fmt.Errorf("帖子信息为空")
	}

	content := buildReplyContent(result)
	imageURLs := collectReplyImageURLs(result)
	resp, err := llm.GenerateResponse(content, imageURLs)
	if err != nil {
		return err
	}

	replyText := firstLLMResponseContent(resp)
	if replyText == "" {
		return fmt.Errorf("LLM 回复为空")
	}

	linkID := result.PostLink.LinkID
	var commentID int64
	if result.RootComment == nil {
		commentID, err = api.CommentPost(saved_sess.HeyboxID, linkID, replyText)
		if err != nil {
			return err
		}
		logReplySuccess(commentID, linkID, replyText, resp.Usage)
		return nil
	}

	rootID := result.RootComment.CommentID
	replyID := rootID
	if result.TargetComment != nil {
		replyID = result.TargetComment.CommentID
	}

	if replyID == rootID {
		commentID, err = api.CommentRoot(saved_sess.HeyboxID, linkID, rootID, replyText)
		if err != nil {
			return err
		}
		logReplySuccess(commentID, linkID, replyText, resp.Usage)
		return nil
	}

	commentID, err = api.CommentReply(saved_sess.HeyboxID, linkID, rootID, replyID, replyText)
	if err != nil {
		return err
	}
	logReplySuccess(commentID, linkID, replyText, resp.Usage)
	return nil
}

func logReplySuccess(commentID, linkID int64, replyText string, usage llm.ChatCompletionUsage) {
	logger.Info(
		"回复成功: comment_id=%d, link_id=%d, content=%q, token消耗: prompt=%d, completion=%d, total=%d, cached=%d, reasoning=%d",
		commentID,
		linkID,
		replyText,
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
		usage.PromptTokensDetails.CachedTokens,
		usage.CompletionTokensDetails.ReasoningTokens,
	)
}

func buildReplyContent(result MessageArrangeResult) string {
	post := result.PostLink
	var b strings.Builder

	writeReplyField(&b, "帖子标题", post.Title)
	writeReplyField(&b, "帖子内容", post.Description)
	writeReplyField(&b, "帖子主题", post.TopicName)
	writeReplyField(&b, "帖子tag", strings.Join(post.ContentTags, "，"))

	if result.RootComment != nil {
		writeReplyField(&b, "根评论内容", result.RootComment.Text)
	}
	if shouldIncludeTargetComment(result.RootComment, result.TargetComment) {
		writeReplyField(&b, "reply评论内容", result.TargetComment.Text)
	}

	return strings.TrimSpace(b.String())
}

func writeReplyField(b *strings.Builder, name, value string) {
	value = strings.TrimSpace(value)
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(name)
	b.WriteString("：")
	b.WriteString(value)
}

func shouldIncludeTargetComment(root, target *api.PostComment) bool {
	return target != nil && root.CommentID != target.CommentID
}

func collectReplyImageURLs(result MessageArrangeResult) []string {
	imageURLs := make([]string, 0)
	seen := make(map[string]struct{})

	if result.PostLink != nil {
		imageURLs = appendUniqueStrings(imageURLs, seen, result.PostLink.ImgURLs)
	}
	if result.RootComment != nil {
		imageURLs = appendUniqueStrings(imageURLs, seen, result.RootComment.ImgURLs)
	}

	return imageURLs
}

func appendUniqueStrings(dst []string, seen map[string]struct{}, values []string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dst = append(dst, value)
	}
	return dst
}

func firstLLMResponseContent(resp *llm.ChatCompletionResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content)
}
