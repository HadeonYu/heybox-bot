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

func init() {
	api.SetCookieUpdateHandler(updateSavedSessionCookies)
}

func newSession(state *api.QRStateResult) *session {
	return &session{
		HeyboxID: state.HeyboxID,
		Nickname: state.Nickname,
		Avatar:   state.Avatar,
		Level:    state.AccountInfo.LevelInfo.Level,
		Cookies:  state.Cookies,
	}
}

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

func sameCookie(a, b *http.Cookie) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && a.Domain == b.Domain && a.Path == b.Path
}

func updateSavedSessionCookies(cookies []*http.Cookie) {
	if saved_sess == nil || len(cookies) == 0 {
		return
	}

	saved_sess.updateCookies(cookies)
	if err := saved_sess.save(); err != nil {
		logger.Error("保存更新后的会话 Cookie 失败: %v", err)
	}
}

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
