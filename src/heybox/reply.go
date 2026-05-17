package heybox

import (
	"fmt"
	"heybox-bot/config"
	"heybox-bot/db"
	"heybox-bot/heybox/api"
	"heybox-bot/llm"
	"heybox-bot/logger"
	"strconv"
	"strings"
)

const (
	whiteListRefusedReply = "抱歉，你不在白名单中"
	frequencyRefusedReply = "请求频率超过了设定值"
	videoUnsupportedReply = "不支持带视频的消息 [cube_沧桑]"
)

// reply 根据整理后的 @ 消息生成回复并发布到对应帖子或评论下。
func reply(results []MessageArrangeResult) error {
	if saved_sess == nil {
		return fmt.Errorf("会话为空")
	}

	for _, result := range results {
		if err := replyOne(result); err != nil {
			if insertErr := insertErrorMessage(result, err); insertErr != nil {
				return fmt.Errorf("记录消息 %d 失败状态失败: %w", result.MessageID, insertErr)
			}
			logger.Error("回复消息 %d 失败: %v", result.MessageID, err)
		}
		if err := saveLastAtMessageTime(result.Timestamp); err != nil {
			return fmt.Errorf("更新消息 %d 处理时间失败: %w", result.MessageID, err)
		}
	}
	return nil
}

func replyOne(result MessageArrangeResult) error {
	refused, err := shouldRefuseService(result.User.UserID)
	if err != nil {
		return err
	}
	if refused {
		reason := refusedReplyText()
		linkID := replyLinkID(result)
		if err := insertRefusedMessage(result, 0, linkID, reason); err != nil {
			return err
		}
		logger.Info("拒绝服务: message_id=%d, user_id=%s, user_name: %s, link_id=%d, reason=%q", result.MessageID, result.User.UserID, result.User.Username, linkID, reason)
		return nil
	}

	if result.HasVideo != 0 {
		commentID, linkID, err := publishReply(result, videoUnsupportedReply)
		if err != nil {
			return err
		}
		if err := insertRefusedMessage(result, commentID, linkID, videoUnsupportedReply); err != nil {
			return err
		}
		logReplySuccess(commentID, linkID, videoUnsupportedReply, llm.ChatCompletionUsage{})
		return nil
	}

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

	commentID, linkID, err := publishReply(result, replyText)
	if err != nil {
		return err
	}
	if err := insertSuccessMessage(result, commentID, linkID, replyText, resp.Usage); err != nil {
		return err
	}
	logReplySuccess(commentID, linkID, replyText, resp.Usage)
	return nil
}

func refusedReplyText() string {
	if config.GetBotMode() == config.BotModeFrequency {
		return frequencyRefusedReply
	}
	return whiteListRefusedReply
}

func replyLinkID(result MessageArrangeResult) int64 {
	if result.LinkID != 0 {
		return result.LinkID
	}
	if result.PostLink != nil {
		return result.PostLink.LinkID
	}
	return 0
}

func publishReply(result MessageArrangeResult, replyText string) (int64, int64, error) {
	linkID := replyLinkID(result)
	if linkID == 0 {
		return 0, 0, fmt.Errorf("帖子 ID 为空")
	}

	if result.IsPost || result.RootCommentID == 0 {
		commentID, err := api.CommentPost(saved_sess.HeyboxID, linkID, replyText)
		return commentID, linkID, err
	}

	rootID := result.RootCommentID
	if rootID == 0 && result.RootComment != nil {
		rootID = result.RootComment.CommentID
	}

	replyID := result.TargetCommentID
	if replyID == 0 && result.TargetComment != nil {
		replyID = result.TargetComment.CommentID
	}
	if replyID == 0 {
		replyID = rootID
	}

	if replyID == rootID {
		commentID, err := api.CommentRoot(saved_sess.HeyboxID, linkID, rootID, replyText)
		return commentID, linkID, err
	}

	commentID, err := api.CommentReply(saved_sess.HeyboxID, linkID, rootID, replyID, replyText)
	return commentID, linkID, err
}

func insertSuccessMessage(result MessageArrangeResult, commentID, linkID int64, replyText string, usage llm.ChatCompletionUsage) error {
	return insertMessage(result, db.MessageStatusSuccess, commentID, linkID, replyText, usage)
}

func insertRefusedMessage(result MessageArrangeResult, commentID, linkID int64, replyText string) error {
	return insertMessage(result, db.MessageStatusRefused, commentID, linkID, replyText, llm.ChatCompletionUsage{})
}

func insertErrorMessage(result MessageArrangeResult, replyErr error) error {
	linkID := result.LinkID
	if result.PostLink != nil && result.PostLink.LinkID != 0 {
		linkID = result.PostLink.LinkID
	}

	commentContent := ""
	if replyErr != nil {
		commentContent = replyErr.Error()
	}
	return insertMessage(result, db.MessageStatusError, 0, linkID, commentContent, llm.ChatCompletionUsage{})
}

func insertMessage(result MessageArrangeResult, status string, commentID, linkID int64, replyText string, usage llm.ChatCompletionUsage) error {
	userID, err := strconv.ParseInt(result.User.UserID, 10, 64)
	if err != nil {
		logger.Warn("解析用户 ID %q 失败: %v", result.User.UserID, err)
	}

	message := &db.Message{
		MessageID:       result.MessageID,
		Status:          status,
		LinkID:          linkID,
		UserID:          userID,
		UserName:        result.User.Username,
		IsPost:          result.IsPost,
		TriggerID:       replyTriggerID(result, linkID),
		TriggerContent:  replyTriggerContent(result),
		CommentID:       commentID,
		CommentContent:  replyText,
		PromptToken:     int64(usage.PromptTokens),
		CompletionToken: int64(usage.CompletionTokens),
		CachedToken:     int64(usage.PromptTokensDetails.CachedTokens),
		ReasoningToken:  int64(usage.CompletionTokensDetails.ReasoningTokens),
		TotalToken:      int64(usage.TotalTokens),
	}
	if err := db.InsertMessage(message); err != nil {
		return err
	}
	return nil
}

func replyTriggerID(result MessageArrangeResult, linkID int64) int64 {
	if result.IsPost {
		return linkID
	}
	if result.TargetCommentID != 0 {
		return result.TargetCommentID
	}
	if result.TargetComment != nil {
		return result.TargetComment.CommentID
	}
	if result.RootCommentID != 0 {
		return result.RootCommentID
	}
	if result.RootComment != nil {
		return result.RootComment.CommentID
	}
	return linkID
}

func replyTriggerContent(result MessageArrangeResult) string {
	if result.IsPost && result.PostLink != nil {
		return result.PostLink.Title
	}
	if result.TargetComment != nil {
		return result.TargetComment.Text
	}
	if result.RootComment != nil {
		return result.RootComment.Text
	}
	return result.TriggerContent
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
	if target == nil {
		return false
	}
	if root == nil {
		return true
	}
	return root.CommentID != target.CommentID
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
