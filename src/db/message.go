package db

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	MessageStatusSuccess = "success"
	MessageStatusError   = "error"
	MessageStatusRefused = "refused"
)

type Message struct {
	MessageID int64 `gorm:"column:message_id;primaryKey;autoIncrement:false;not null" json:"message_id"`

	Status string `gorm:"column:status;type:text;not null" json:"status"`

	LinkID   int64  `gorm:"column:link_id;not null" json:"link_id"`
	UserID   int64  `gorm:"column:user_id;not null" json:"user_id"`
	UserName string `gorm:"column:user_name;not null" json:"user_name"`

	// true: 帖子触发
	// false: 评论触发
	IsPost bool `gorm:"column:is_post;not null" json:"is_post"`

	// 触发源 ID：
	// 如果 IsPost=true，可以理解为帖子 ID；
	// 如果 IsPost=false，可以理解为触发评论 ID。
	TriggerID int64 `gorm:"column:trigger_id;not null" json:"trigger_id"`

	// 触发内容：
	// 帖子标题或触发机器人回复的评论内容
	TriggerContent string `gorm:"column:trigger_content;type:text;not null" json:"trigger_content"`

	// 机器人回复后生成的评论 ID
	CommentID int64 `gorm:"column:comment_id;not null" json:"comment_id"`

	// 机器人最终发布出去的评论内容
	CommentContent string `gorm:"column:comment_content;type:text;not null" json:"comment_content"`

	PromptToken     int64 `gorm:"column:prompt_token;not null" json:"prompt_token"`
	CompletionToken int64 `gorm:"column:completion_token;not null" json:"completion_token"`
	CachedToken     int64 `gorm:"column:cached_token;not null" json:"cached_token"`
	ReasoningToken  int64 `gorm:"column:reasoning_token;not null" json:"reasoning_token"`
	TotalToken      int64 `gorm:"column:total_token;not null" json:"total_token"`
}

func (Message) TableName() string {
	return "message"
}

func initMessage(db *gorm.DB) error {
	if err := db.AutoMigrate(&Message{}); err != nil {
		return fmt.Errorf("初始化 message 表失败: %w", err)
	}
	return nil
}

// InsertMessage 插入一条消息处理记录。
func InsertMessage(message *Message) error {
	if message == nil {
		return fmt.Errorf("消息记录为空")
	}

	db, err := getDatabase()
	if err != nil {
		return err
	}
	if err := db.Create(message).Error; err != nil {
		return fmt.Errorf("插入 message 记录失败: %w", err)
	}
	return nil
}

// DeleteMessage 按 message_id 删除一条消息处理记录。
func DeleteMessage(messageID int64) error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	if err := db.Delete(&Message{}, "message_id = ?", messageID).Error; err != nil {
		return fmt.Errorf("删除 message 记录失败: %w", err)
	}
	return nil
}
