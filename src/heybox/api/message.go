package api

import (
	"encoding/json"
	"fmt"
	"heybox-bot/logger"
	"strconv"
)

const (
	MessageTypeAtPost    = 16
	MessageTypeAtComment = 17
	MessageNumLimit      = 20
)

type MessageListResult struct {
	Messages []Message `json:"messages"`
}

type User struct {
	Username string `json:"username"`
	UserID   string `json:"userid"`
}

type Message struct {
	MessageID     int64  `json:"message_id"`
	User          User   `json:"user_a"`
	Timestamp     string `json:"timestamp"`
	CommentID     int64  `json:"comment_a_id"`    // @我的评论的id
	RootCommentID int64  `json:"root_comment_id"` // 如果@我的评论就是根评论，这个值和CommentID一样
	LinkID        int64  `json:"linkid"`
	HasVideo      int    `json:"has_video"`
	Text          string `json:"comment_a_text"`
	MessageType   int    `json:"message_type"`
	IsPost        bool   `json:"-"`
}

// UnmarshalJSON 将消息中心响应解析为统一的 Message 结构。
func (m *Message) UnmarshalJSON(data []byte) error {
	type messageAlias Message
	var aux struct {
		messageAlias
		Link struct {
			LinkID   int64  `json:"linkid"`
			HasVideo int    `json:"has_video"`
			Text     string `json:"description"`
		} `json:"link"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*m = Message(aux.messageAlias)
	m.IsPost = m.MessageType == MessageTypeAtPost
	if m.IsPost {
		m.LinkID = aux.Link.LinkID
		m.HasVideo = aux.Link.HasVideo
		m.Text = aux.Link.Text
	}

	return nil
}

// GetAtMessage 获取@我的消息
func GetAtMessage(heyboxID string, offset int) ([]Message, error) {
	resp, err := GetRequest("/bbs/app/user/message", heyboxID, map[string]string{
		"message_type": strconv.Itoa(MessageTypeAtPost),
		"app":          "heybox",
		"offset":       strconv.Itoa(offset),
		"limit":        strconv.Itoa(MessageNumLimit),
		"no_more":      "false",
	})
	if err != nil {
		err = fmt.Errorf("执行 HTTP 请求失败: %w", err)
		logger.Error("%v", err)
		return nil, err
	}

	if resp.Status != "ok" {
		err := fmt.Errorf("获取消息失败，响应状态: %v，消息: %v", resp.Status, resp.Msg)
		logger.Error("%v", err)
		return nil, err
	}

	var result MessageListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("解析消息结果失败: %w", err)
	}

	return result.Messages, nil
}
