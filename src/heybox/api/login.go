package api

import (
	"encoding/json"
	"fmt"
	"heybox-bot/logger"
	"net/http"
	"net/url"
	"time"
)

type QRCodeResult struct {
	QRURL  string        `json:"qr_url"`
	Expire time.Duration `json:"expire"`
	QRID   string        `json:"-"`
}

type QRStateResult struct {
	Error       string         `json:"error"`
	ErrorMsg    string         `json:"error_msg"`
	HeyboxID    string         `json:"heyboxid"`
	Avatar      string         `json:"avatar"`
	Nickname    string         `json:"nickname"`
	AccountInfo AccountDetail  `json:"account_detail"`
	Cookies     []*http.Cookie `json:"-"`
}

type AccountDetail struct {
	Username         string           `json:"username"`
	UserID           string           `json:"userid"`
	AvatarDecoration AvatarDecoration `json:"avatar_decoration"`
	Avatar           string           `json:"avatar"`
	Avartar          string           `json:"avartar"`
	LevelInfo        LevelInfo        `json:"level_info"`
}

type AvatarDecoration struct {
	SrcType string `json:"src_type"`
	SrcURL  string `json:"src_url"`
}

type LevelInfo struct {
	Level int `json:"level"`
}

// GetQRCode 解析QR码的获取结果，返回结果, 过期时间
func GetQRCode() (*QRCodeResult, error) {
	resp, err := GetRequest("/account/get_qrcode_url/", "")
	if err != nil {
		return nil, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("获取二维码失败，状态: %v，消息: %v", resp.Status, resp.Msg)
	}

	var result QRCodeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("解析二维码结果失败: %w", err)
	}

	u, err := url.Parse(result.QRURL)
	if err != nil {
		return nil, fmt.Errorf("解析二维码地址失败: %w", err)
	}

	qrID := u.Query().Get("qr")
	if qrID == "" {
		return nil, fmt.Errorf("二维码地址中缺少 qr 参数: %v", result.QRURL)
	}

	result.QRID = qrID
	result.Expire = time.Duration(int64(result.Expire)) * time.Second

	return &result, nil
}

// GetQRCodeState 获取二维码状态：wait, ready, ok和其他错误状态
func GetQRCodeState(qrID string) (*QRStateResult, error) {
	resp, err := GetRequest("/account/qr_state/", "", map[string]string{"qr": qrID})
	if err != nil {
		err = fmt.Errorf("执行 HTTP 请求失败: %w", err)
		logger.Error("%v", err)
		return nil, err
	}

	if resp.Status != "ok" {
		return nil, fmt.Errorf("获取二维码状态失败，响应状态: %v，消息: %v", resp.Status, resp.Msg)
	}

	var result QRStateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("解析二维码状态失败: %w", err)
	}
	result.Cookies = resp.Cookies
	return &result, nil
}

// GetUserPermission 获取用户权限，仅用于验证session信息
func GetUserPermission(heyboxID string) (bool, []*http.Cookie, error) {
	resp, err := GetRequest("/bbs/app/api/user/permission", heyboxID)
	if err != nil {
		return false, nil, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}

	switch resp.Status {
	case "":
		if !isUserPermissionResponse(resp.Raw) {
			return false, resp.Cookies, fmt.Errorf("用户权限响应异常: %s", string(resp.Raw))
		}
		return true, resp.Cookies, nil
	case "ok":
		return true, resp.Cookies, nil
	case "login", "relogin":
		return false, resp.Cookies, nil

	default:
		return false, nil, fmt.Errorf("用户权限校验失败，状态=%s，消息=%s", resp.Status, resp.Msg)
	}
}

// isUserPermissionResponse 因为GetUserPermission返回结构可能和通用结构不一致，这个函数用来单独判断返回知是否成功
func isUserPermissionResponse(raw json.RawMessage) bool {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}

	_, ok := body["visitor_enabled"] // 认为有这个字段就算成功
	return ok
}
