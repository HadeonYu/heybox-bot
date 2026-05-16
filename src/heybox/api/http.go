package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"heybox-bot/logger"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const apiBaseURL = "https://api.xiaoheihe.cn"

var jar, _ = cookiejar.New(nil)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Jar:     jar,
}

var (
	cookieUpdateHandler func([]*http.Cookie)
	deviceID            string
	deviceIDMu          sync.RWMutex
	xhhTokenID          string
	xhhTokenIDMu        sync.RWMutex
)

type response struct {
	Status  string          `json:"status"`
	Msg     string          `json:"msg"`
	Version string          `json:"version"`
	Result  json.RawMessage `json:"result"`
	Cookies []*http.Cookie  `json:"-"`
	Raw     json.RawMessage `json:"-"`
}

// SetCookies 将已有 Cookie 写入 HTTP 客户端的 CookieJar。
func SetCookies(cookies []*http.Cookie) {
	if len(cookies) == 0 {
		return
	}

	u, _ := url.Parse(apiBaseURL + "/")
	httpClient.Jar.SetCookies(u, cookies)
}

// SetCookieUpdateHandler 设置响应 Cookie 更新时的回调函数。
func SetCookieUpdateHandler(handler func([]*http.Cookie)) {
	cookieUpdateHandler = handler
}

// SetDeviceID 设置后续 API 请求使用的设备 ID。
func SetDeviceID(id string) {
	deviceIDMu.Lock()
	defer deviceIDMu.Unlock()
	deviceID = id
}

// getDeviceID 返回当前 API 请求使用的设备 ID。
func getDeviceID() string {
	deviceIDMu.RLock()
	defer deviceIDMu.RUnlock()
	return deviceID
}

// SetXHHTokenID 设置后续 API 请求 Cookie 中使用的 x_xhh_tokenid。
func SetXHHTokenID(tokenID string) {
	xhhTokenIDMu.Lock()
	defer xhhTokenIDMu.Unlock()
	xhhTokenID = tokenID
}

// getXHHTokenID 返回当前 API 请求使用的 x_xhh_tokenid。
func getXHHTokenID() string {
	xhhTokenIDMu.RLock()
	defer xhhTokenIDMu.RUnlock()
	return xhhTokenID
}

// GetRequest 发起带通用参数的小黑盒 GET 请求。
func GetRequest(apiPath string, heyboxID string, extraQuery ...map[string]string) (*response, error) {
	return request(http.MethodGet, apiPath, heyboxID, extraQuery...)
}

// PostRequest 发起带通用参数和表单数据的小黑盒 POST 请求。
func PostRequest(apiPath string, heyboxID string, form ...map[string]string) (*response, error) {
	return request(http.MethodPost, apiPath, heyboxID, form...)
}

// request 构造签名参数并发送小黑盒 API 请求。
func request(method string, apiPath string, heyboxID string, values ...map[string]string) (*response, error) {
	hkey, ts, nonce := makeHeyboxSign(apiPath)

	q := url.Values{}
	q.Set("os_type", "web")
	q.Set("app", "web")
	q.Set("client_type", "web")
	q.Set("version", "999.0.4")
	q.Set("web_version", "2.5")
	q.Set("x_client_type", "web")
	q.Set("x_app", "heybox_website")
	q.Set("heybox_id", heyboxID)
	q.Set("x_os_type", "Windows")
	q.Set("device_info", "Chrome")
	q.Set("device_id", getDeviceID())
	q.Set("hkey", hkey)
	q.Set("_time", fmt.Sprintf("%d", ts))
	q.Set("nonce", nonce)

	var body io.Reader
	if method == http.MethodGet {
		setValues(q, values...)
	} else {
		form := url.Values{}
		setValues(form, values...)
		body = strings.NewReader(form.Encode())
	}

	reqURL := apiBaseURL + apiPath + "?" + q.Encode()
	logger.Debug("request url: %v", reqURL)

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}

	setDefaultHeaders(req)
	if tokenID := getXHHTokenID(); tokenID != "" {
		req.AddCookie(&http.Cookie{
			Name:  "x_xhh_tokenid",
			Value: tokenID,
		})
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("小黑盒 API 请求失败: %s", resp.Status)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	// logger.Debug("resp: %+v", string(bodyBytes))
	var result response
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&result); err != nil {
		return nil, err
	}
	result.Raw = bodyBytes
	result.Cookies = resp.Cookies()
	if len(result.Cookies) > 0 && cookieUpdateHandler != nil {
		cookieUpdateHandler(result.Cookies)
	}

	return &result, nil
}

// setDefaultHeaders 为请求设置小黑盒网页端默认请求头。
func setDefaultHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Origin", "https://www.xiaoheihe.cn")
	req.Header.Set("Referer", "https://www.xiaoheihe.cn/")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="147", "Not.A/Brand";v="8", "Chromium";v="147"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36")
}

// setValues 将额外键值写入 URL 或表单参数集合。
func setValues(values url.Values, extraValues ...map[string]string) {
	for _, extra := range extraValues {
		for k, v := range extra {
			values.Set(k, v)
		}
	}
}
