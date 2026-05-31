package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const latestVersionInfoURL = "https://assets.yuhd.site/heybox-bot/latest/version.json"

// VersionInfo 表示 latest/version.json 的内容。
type VersionInfo struct {
	Version string                 `json:"version"`
	Files   map[string]VersionFile `json:"files"`
}

// VersionFile 表示一个平台发布包的信息。
type VersionFile struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// fetch 拉取并解析最新版本信息
func fetch() (VersionInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(latestVersionInfoURL)
	if err != nil {
		return VersionInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return VersionInfo{}, fmt.Errorf("http请求失败，状态: %d, %s", resp.StatusCode, resp.Status)
	}

	var info VersionInfo
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024))
	if err := decoder.Decode(&info); err != nil {
		return VersionInfo{}, fmt.Errorf("解析版本信息失败: %w", err)
	}

	if err := validateVersionInfo(info); err != nil {
		return VersionInfo{}, err
	}
	return info, nil
}

func downloadFile(rawURL string, targetPath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("下载发行版失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载发行版失败，状态: %d, %s", resp.StatusCode, resp.Status)
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建下载文件失败: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入下载文件失败: %w", err)
	}
	return nil
}

func verifySHA256(path string, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开下载文件失败: %w", err)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return fmt.Errorf("计算 sha256 失败: %w", err)
	}

	actual := hex.EncodeToString(digest.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expected)) {
		return fmt.Errorf("sha256 校验失败: expected %s, got %s", expected, actual)
	}
	return nil
}

func validateVersionInfo(info VersionInfo) error {
	if strings.TrimSpace(info.Version) == "" {
		return fmt.Errorf("版本号为空")
	}
	if extractVersion(info.Version) == "" {
		return fmt.Errorf("版本号格式错误: %q", info.Version)
	}
	if len(info.Files) == 0 {
		return fmt.Errorf("版本文件列表为空")
	}

	for name, file := range info.Files {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("版本文件平台名为空")
		}
		if strings.TrimSpace(file.URL) == "" {
			return fmt.Errorf("%s 下载地址为空", name)
		}
		if strings.TrimSpace(file.SHA256) == "" {
			return fmt.Errorf("%s sha256 为空", name)
		}
	}
	return nil
}
