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
)

type response struct {
	Status  string          `json:"status"`
	Msg     string          `json:"msg"`
	Version string          `json:"version"`
	Result  json.RawMessage `json:"result"`
	Cookies []*http.Cookie  `json:"-"`
	Raw     json.RawMessage `json:"-"`
}

func SetCookies(cookies []*http.Cookie) {
	if len(cookies) == 0 {
		return
	}

	u, _ := url.Parse(apiBaseURL + "/")
	httpClient.Jar.SetCookies(u, cookies)
}

func SetCookieUpdateHandler(handler func([]*http.Cookie)) {
	cookieUpdateHandler = handler
}

func SetDeviceID(id string) {
	deviceIDMu.Lock()
	defer deviceIDMu.Unlock()
	deviceID = id
}

func getDeviceID() string {
	deviceIDMu.RLock()
	defer deviceIDMu.RUnlock()
	return deviceID
}

func GetRequest(apiPath string, heyboxID string, extraQuery ...map[string]string) (*response, error) {
	return request(http.MethodGet, apiPath, heyboxID, extraQuery...)
}

func PostRequest(apiPath string, heyboxID string, form ...map[string]string) (*response, error) {
	return request(http.MethodPost, apiPath, heyboxID, form...)
}

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

func setValues(values url.Values, extraValues ...map[string]string) {
	for _, extra := range extraValues {
		for k, v := range extra {
			values.Set(k, v)
		}
	}
}
