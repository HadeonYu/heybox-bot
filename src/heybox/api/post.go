package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const PostTreeLimit = 20

type PostTreeResult struct {
	TotalPage     int             `json:"total_page"`
	HasMoreFloors int             `json:"has_more_floors"`
	Link          PostLink        `json:"link"`
	Comments      [][]PostComment `json:"comments"`
}

type PostLink struct {
	LinkID      int64    `json:"linkid"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ImgURLs     []string `json:"img_urls"`
	TopicName   string   `json:"topic_name"`
	ContentTags []string `json:"content_tags"`
}

func (r *PostTreeResult) UnmarshalJSON(data []byte) error {
	type resultAlias PostTreeResult
	var aux struct {
		resultAlias
		Comments []commentGroup `json:"comments"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*r = PostTreeResult(aux.resultAlias)
	r.Comments = collectCommentGroups(aux.Comments)
	return nil
}

func (l *PostLink) UnmarshalJSON(data []byte) error {
	type linkAlias PostLink
	var aux struct {
		linkAlias
		Topics []struct {
			Name string `json:"name"`
		} `json:"topics"`
		ContentTags []struct {
			Text string `json:"text"`
		} `json:"content_tags"`
		Text string `json:"text"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*l = PostLink(aux.linkAlias)
	if len(aux.Topics) > 0 {
		l.TopicName = aux.Topics[0].Name
	}
	if aux.Text != "" {
		l.parseTextContent(aux.Text)
	}
	for _, tag := range aux.ContentTags {
		if tag.Text != "" {
			l.ContentTags = append(l.ContentTags, tag.Text)
		}
	}

	return nil
}

func (l *PostLink) parseTextContent(text string) {
	var contents []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		URL  string `json:"url"`
	}

	if err := json.Unmarshal([]byte(text), &contents); err != nil {
		return
	}

	for _, content := range contents {
		switch content.Type {
		case "text":
			if content.Text != "" {
				l.Description = content.Text
			}
		case "img":
			if content.URL != "" {
				l.ImgURLs = append(l.ImgURLs, content.URL)
			}
		}
	}
}

// GetPostTree 拉取帖子内容和评论树
func GetPostTree(heyboxID string, linkID int64, page int) (*PostTreeResult, error) {
	if page < 1 {
		return nil, fmt.Errorf("页码必须大于等于 1")
	}

	isFirst := "0"
	if page == 1 {
		isFirst = "1"
	}

	resp, err := GetRequest("/bbs/app/link/tree", heyboxID, map[string]string{
		"link_id":    strconv.FormatInt(linkID, 10),
		"page":       strconv.Itoa(page),
		"is_first":   isFirst,
		"index":      "1",
		"limit":      strconv.Itoa(PostTreeLimit),
		"owner_only": "0",
	})
	if err != nil {
		return nil, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("获取帖子评论树失败，响应状态: %v，消息: %v", resp.Status, resp.Msg)
	}

	var result PostTreeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("解析帖子评论树结果失败: %w", err)
	}

	return &result, nil
}
