package heybox

import (
	"encoding/json"
	"fmt"
	"heybox-bot/heybox/api"
	"heybox-bot/logger"
	"net/http"
)

type session struct {
	HeyboxID string         `json:"heybox_id,omitempty"`
	Nickname string         `json:"nickname,omitempty"`
	Avatar   string         `json:"avatar,omitempty"`
	Level    int            `json:"level,omitempty"`
	Cookies  []*http.Cookie `json:"cookies"`
}

// init 注册 API Cookie 更新回调。
func init() {
	api.SetCookieUpdateHandler(updateSavedSessionCookies)
}

// newSession 根据二维码登录结果创建会话对象。
func newSession(state *api.QRStateResult) *session {
	return &session{
		HeyboxID: state.HeyboxID,
		Nickname: state.Nickname,
		Avatar:   state.Avatar,
		Level:    state.AccountInfo.LevelInfo.Level,
		Cookies:  state.Cookies,
	}
}

// save 将当前会话保存到元数据目录。
func (s *session) save() error {
	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Errorf("序列化会话失败: %w", err)
	}

	if err := writeMetadataFile(sessionFile, b, 0o600); err != nil {
		return fmt.Errorf("写入会话文件失败: %w", err)
	}

	return nil
}

// updateCookies 将响应中的 Cookie 合并到当前会话。
func (s *session) updateCookies(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}

		replaced := false
		for i, savedCookie := range s.Cookies {
			if sameCookie(savedCookie, cookie) {
				s.Cookies[i] = cookie
				replaced = true
				break
			}
		}

		if !replaced {
			s.Cookies = append(s.Cookies, cookie)
		}
	}
}

// sameCookie 判断两个 Cookie 是否代表同一个存储项。
func sameCookie(a, b *http.Cookie) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && a.Domain == b.Domain && a.Path == b.Path
}

// updateSavedSessionCookies 更新全局会话中的 Cookie 并持久化。
func updateSavedSessionCookies(cookies []*http.Cookie) {
	if saved_sess == nil || len(cookies) == 0 {
		return
	}

	saved_sess.updateCookies(cookies)
	if err := saved_sess.save(); err != nil {
		logger.Error("保存更新后的会话 Cookie 失败: %v", err)
	}
}

// loadSession 从元数据目录读取并恢复会话。
func loadSession() (*session, error) {
	b, fromLegacy, err := readMetadataFile(sessionFile, legacySessionFile)
	if err != nil {
		return nil, fmt.Errorf("读取会话文件失败: %w", err)
	}

	var s session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("解析会话失败: %w", err)
	}

	api.SetCookies(s.Cookies)
	if fromLegacy {
		if err := s.save(); err != nil {
			return nil, err
		}
	}

	return &s, nil
}
