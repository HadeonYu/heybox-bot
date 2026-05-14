package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type PostComment struct {
	CommentID int64    `json:"commentid"`
	Text      string   `json:"text"`
	ImgURLs   []string `json:"img_urls"`
}

type commentImage struct {
	URL string `json:"url"`
}

type commentGroup struct {
	Comments []PostComment `json:"comment"`
}

type SubCommentsResult struct {
	HasMore  bool          `json:"has_more"`
	LastVal  int64         `json:"lastval"`
	Comments []PostComment `json:"comments"`
}

type createCommentResponse struct {
	CommentID int64 `json:"commentid"`
}

func (c *PostComment) UnmarshalJSON(data []byte) error {
	type commentAlias PostComment
	var aux struct {
		commentAlias
		Imgs []commentImage `json:"imgs"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*c = PostComment(aux.commentAlias)
	for _, img := range aux.Imgs {
		if img.URL != "" {
			c.ImgURLs = append(c.ImgURLs, img.URL)
		}
	}

	return nil
}

func collectCommentGroups(groups []commentGroup) [][]PostComment {
	comments := make([][]PostComment, 0, len(groups))
	for _, group := range groups {
		comments = append(comments, group.Comments)
	}
	return comments
}

// GetSubComments 拉取某个根评论下的更多子评论。
func GetSubComments(heyboxID string, rootCommentID, lastVal int64) (*SubCommentsResult, error) {
	resp, err := GetRequest("/bbs/app/comment/sub/comments", heyboxID, map[string]string{
		"lastval":         strconv.FormatInt(lastVal, 10),
		"root_comment_id": strconv.FormatInt(rootCommentID, 10),
	})
	if err != nil {
		return nil, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("获取子评论失败，响应状态: %v，消息: %v", resp.Status, resp.Msg)
	}

	var result SubCommentsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("解析子评论结果失败: %w", err)
	}

	return &result, nil
}

// CreateComment 创建评论，并返回服务端生成的 commentid。回复帖子时，rootID和replyID都是-1
func CreateComment(heyboxID string, linkID, rootID, replyID int64, text string) (int64, error) {
	resp, err := PostRequest("/bbs/app/comment/create", heyboxID, map[string]string{
		"link_id":  strconv.FormatInt(linkID, 10),
		"reply_id": strconv.FormatInt(replyID, 10),
		"root_id":  strconv.FormatInt(rootID, 10),
		"text":     text,
	})
	if err != nil {
		return 0, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}

	if resp.Status != "ok" {
		return 0, fmt.Errorf("创建评论失败，响应状态: %v，消息: %v", resp.Status, resp.Msg)
	}

	var result createCommentResponse
	if err := json.Unmarshal(resp.Raw, &result); err != nil {
		return 0, fmt.Errorf("解析创建评论响应失败: %w", err)
	}
	if result.CommentID == 0 {
		return 0, fmt.Errorf("创建评论返回的 commentid 为空")
	}

	return result.CommentID, nil
}
