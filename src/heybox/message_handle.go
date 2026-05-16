package heybox

import (
	"encoding/json"
	"fmt"
	"heybox-bot/config"
	"heybox-bot/heybox/api"
	"heybox-bot/logger"
	"html"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	atMessageTimerName = "getAtMsg"
	messageFetchSleep  = 1 * time.Second
)

var heyboxMentionRe = regexp.MustCompile(`(?is)<a\b[^>]*\bdata-user-id\s*=\s*["'][^"']+["'][^>]*>(.*?)</a>\s*`)

type MessageArrangeResult struct {
	MessageID     int64            `json:"message_id"`
	User          api.User         `json:"user"`
	Timestamp     time.Time        `json:"timestamp"`
	HasVideo      int              `json:"has_video"`
	PostLink      *api.PostLink    `json:"post_link,omitempty"`
	RootComment   *api.PostComment `json:"root_comment,omitempty"`
	TargetComment *api.PostComment `json:"target_comment,omitempty"`
}

// getAtMessageCb 定时拉取并整理未读 @ 消息。
func getAtMessageCb(timer *TimerContext) error {
	lastTimestamp, err := loadLastAtMessageTimestamp()
	if err != nil {
		return err
	}

	unread, maxTimestamp, err := getUnreadAtMessages(lastTimestamp)
	if err != nil {
		return err
	}

	if maxTimestamp > lastTimestamp {
		if err := saveLastAtMessageTimestamp(maxTimestamp); err != nil {
			return err
		}
	}

	if len(unread) == 0 {
		nextInterval := nextAtMessageInterval(timer.Interval())
		timer.SetInterval(nextInterval)
		logger.Debug("没有未读 @ 消息，下次检查间隔: %.3f秒", nextInterval.Seconds())
		return nil
	}

	timer.SetInterval(config.GetBotInitWaitTime())
	results := make([]MessageArrangeResult, 0, len(unread))
	for i, msg := range unread {
		result, err := arrangeUnreadAtMessage(msg)
		if err != nil {
			logger.Error("整理未读 @ 消息 %d 失败: %v", msg.MessageID, err)
		} else {
			results = append(results, result)
		}
		if i < len(unread)-1 {
			time.Sleep(messageFetchSleep)
		}
	}
	logger.Info("已整理 %d 条未读 @ 消息", len(results))
	reply(results)
	return nil
}

// nextAtMessageInterval 根据当前间隔计算带随机波动的下一次检查间隔。
func nextAtMessageInterval(current time.Duration) time.Duration {
	initInterval := config.GetBotInitWaitTime()
	maxInterval := config.GetBotMaxWaitTime()
	nextInterval := min(current*2, maxInterval)
	jitter := nextInterval / 5
	if jitter > 0 {
		nextInterval += time.Duration(rand.Int63n(int64(jitter)*2+1)) - jitter
	}

	if nextInterval < initInterval {
		return initInterval
	}
	if nextInterval > maxInterval {
		return maxInterval
	}
	return nextInterval
}

// getUnreadAtMessages 拉取时间戳晚于已记录时间的未读 @ 消息。
func getUnreadAtMessages(lastTimestamp float64) ([]api.Message, float64, error) {
	if saved_sess == nil {
		return nil, lastTimestamp, fmt.Errorf("会话为空")
	}

	unread := make([]api.Message, 0)
	maxTimestamp := lastTimestamp
	offset := 0

	for {
		msgList, err := api.GetAtMessage(saved_sess.HeyboxID, offset)
		if err != nil {
			return nil, maxTimestamp, fmt.Errorf("获取 @ 消息失败: %w", err)
		}
		if len(msgList) == 0 {
			break
		}

		pageUnread := 0
		for _, msg := range msgList {
			ts, err := parseMessageTimestamp(msg.Timestamp)
			if err != nil {
				logger.Error("解析消息时间戳 %q 失败: %v", msg.Timestamp, err)
				continue
			}

			if ts > maxTimestamp {
				maxTimestamp = ts
			}
			if ts > lastTimestamp {
				unread = append(unread, msg)
				pageUnread++
			}
		}

		if pageUnread != len(msgList) || len(msgList) < api.MessageNumLimit {
			break
		}
		offset += api.MessageNumLimit
	}

	return unread, maxTimestamp, nil
}

// arrangeUnreadAtMessage 根据消息类型整理帖子、根评论和目标评论信息。
func arrangeUnreadAtMessage(msg api.Message) (MessageArrangeResult, error) {
	messageTime, err := parseMessageTime(msg.Timestamp)
	if err != nil {
		return MessageArrangeResult{}, fmt.Errorf("解析消息时间戳 %q 失败: %w", msg.Timestamp, err)
	}

	result := MessageArrangeResult{
		MessageID: msg.MessageID,
		User:      msg.User,
		Timestamp: messageTime,
		HasVideo:  msg.HasVideo,
	}
	defer normalizeMessageArrangeResultMentions(&result)

	if msg.HasVideo != 0 {
		return result, nil
	}

	if msg.IsPost {
		tree, err := fetchPostTreePage(msg.LinkID, 1)
		if err != nil {
			return result, err
		}
		result.PostLink = &tree.Link
		return result, nil
	}

	root, group, tree, err := findRootComment(msg)
	if err != nil {
		return result, err
	}
	result.PostLink = &tree.Link
	result.RootComment = root

	if msg.RootCommentID == msg.CommentID {
		result.TargetComment = root
		return result, nil
	}

	if target := findCommentInGroup(group, msg.CommentID); target != nil {
		result.TargetComment = target
		return result, nil
	}

	target, err := findSubComment(msg, group)
	if err != nil {
		return result, err
	}
	result.TargetComment = target
	return result, nil
}

// findRootComment 分页查找消息对应的根评论和评论分支。
func findRootComment(msg api.Message) (*api.PostComment, []api.PostComment, *api.PostTreeResult, error) {
	for page := 1; ; page++ {
		tree, err := fetchPostTreePage(msg.LinkID, page)
		if err != nil {
			return nil, nil, nil, err
		}

		for _, group := range tree.Comments {
			if len(group) == 0 {
				continue
			}
			if group[0].CommentID == msg.RootCommentID {
				return &group[0], group, tree, nil
			}
		}

		if page >= tree.TotalPage || tree.HasMoreFloors == 0 {
			break
		}
	}

	return nil, nil, nil, fmt.Errorf("帖子 %d 中未找到根评论 %d", msg.LinkID, msg.RootCommentID)
}

// fetchPostTreePage 拉取指定帖子的指定页评论树。
func fetchPostTreePage(linkID int64, page int) (*api.PostTreeResult, error) {
	if saved_sess == nil {
		return nil, fmt.Errorf("会话为空")
	}

	time.Sleep(messageFetchSleep)
	return api.GetPostTree(saved_sess.HeyboxID, linkID, page)
}

// findCommentInGroup 在评论分支中查找指定评论 ID 的评论。
func findCommentInGroup(group []api.PostComment, commentID int64) *api.PostComment {
	for i := range group {
		if group[i].CommentID == commentID {
			return &group[i]
		}
	}
	return nil
}

// findSubComment 继续拉取子评论直到找到消息对应的目标评论。
func findSubComment(msg api.Message, group []api.PostComment) (*api.PostComment, error) {
	if saved_sess == nil {
		return nil, fmt.Errorf("会话为空")
	}
	if len(group) == 0 {
		return nil, fmt.Errorf("根评论 %d 的评论组为空", msg.RootCommentID)
	}

	lastVal := group[len(group)-1].CommentID

	for {
		time.Sleep(messageFetchSleep)

		resp, err := api.GetSubComments(
			saved_sess.HeyboxID,
			msg.RootCommentID,
			lastVal,
		)
		if err != nil {
			return nil, err
		}

		if target := findCommentInGroup(resp.Comments, msg.CommentID); target != nil {
			return target, nil
		}

		if !resp.HasMore || len(resp.Comments) == 0 {
			break
		}
		if resp.LastVal == 0 || resp.LastVal == lastVal {
			lastVal = resp.Comments[len(resp.Comments)-1].CommentID
		} else {
			lastVal = resp.LastVal
		}
	}

	return nil, fmt.Errorf("根评论 %d 下未找到评论 %d", msg.RootCommentID, msg.CommentID)
}

// PlainHeyboxMentionText 将小黑盒 @ 链接文本转换为普通 @ 文本。
func PlainHeyboxMentionText(content string) string {
	content = normalizeEscapedHeyboxMentionText(content)
	matches := heyboxMentionRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	var b strings.Builder
	lastEnd := 0
	for _, match := range matches {
		b.WriteString(content[lastEnd:match[0]])

		mention := html.UnescapeString(strings.TrimSpace(content[match[2]:match[3]]))
		b.WriteString(mention)
		if match[1] < len(content) {
			b.WriteByte(' ')
		}

		lastEnd = match[1]
	}
	b.WriteString(content[lastEnd:])
	return b.String()
}

// normalizeMessageArrangeResultMentions 统一整理结果中的 @ 链接显示文本。
func normalizeMessageArrangeResultMentions(result *MessageArrangeResult) {
	if result.PostLink != nil {
		result.PostLink.Description = PlainHeyboxMentionText(result.PostLink.Description)
		result.PostLink.ImgURLs = limitStrings(result.PostLink.ImgURLs, config.GetBotMaxPostImageNum())
	}
	if result.RootComment != nil {
		result.RootComment.Text = PlainHeyboxMentionText(result.RootComment.Text)
		result.RootComment.ImgURLs = limitStrings(result.RootComment.ImgURLs, config.GetBotMaxCommentImageNum())
	}
	if result.TargetComment != nil {
		result.TargetComment.Text = PlainHeyboxMentionText(result.TargetComment.Text)
	}
}

// limitStrings 将字符串切片限制在指定最大长度内。
func limitStrings(values []string, limit int) []string {
	if limit < 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

// BuildHeyboxMentionText 根据用户 ID 和昵称生成小黑盒 @ 链接文本。
func BuildHeyboxMentionText(userID int64, username string) string {
	username = strings.TrimPrefix(username, "@")
	userIDStr := strconv.FormatInt(userID, 10)
	payload, _ := json.Marshal(struct {
		ProtocolType string `json:"protocol_type"`
		UserID       string `json:"user_id"`
	}{
		ProtocolType: "openUser",
		UserID:       userIDStr,
	})

	href := "https://api.xiaoheihe.cn/open_inapp/#heybox://" + url.QueryEscape(string(payload))
	return fmt.Sprintf(
		`<a data-user-id="%s" href="%s" target="_blank">@%s</a> `,
		html.EscapeString(userIDStr),
		html.EscapeString(href),
		html.EscapeString(username),
	)
}

// normalizeEscapedHeyboxMentionText 将转义后的 HTML 片段还原为可匹配文本。
func normalizeEscapedHeyboxMentionText(content string) string {
	replacer := strings.NewReplacer(
		`\u003c`, "<",
		`\u003C`, "<",
		`\u003e`, ">",
		`\u003E`, ">",
		`\u0026`, "&",
		`\u0027`, "'",
		`\u003d`, "=",
		`\"`, `"`,
	)
	return replacer.Replace(content)
}

// parseMessageTimestamp 将消息时间戳字符串解析为浮点秒数。
func parseMessageTimestamp(timestamp string) (float64, error) {
	ts, err := strconv.ParseFloat(timestamp, 64)
	if err != nil {
		return 0, err
	}
	return ts, nil
}

// parseMessageTime 将消息时间戳字符串解析为 time.Time。
func parseMessageTime(timestamp string) (time.Time, error) {
	parts := strings.SplitN(timestamp, ".", 2)
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	var nsec int64
	if len(parts) == 2 {
		frac := parts[1]
		if len(frac) > 9 {
			frac = frac[:9]
		}
		for len(frac) < 9 {
			frac += "0"
		}
		nsec, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}

	return time.Unix(sec, nsec), nil
}
