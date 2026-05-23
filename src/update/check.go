package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const latestVersionURL = "https://assets.yuhd.site/heybox-bot/latest/VERSION"

var versionPattern = regexp.MustCompile(`v?\d+(?:\.\d+)*`)

// CheckResult 表示一次更新检查的结果。
type CheckResult struct {
	LatestVersion  string
	CurrentVersion string
	LegacyVersion  bool
	NeedUpdate     bool
}

// Check 检查当前 heybox-bot 是否需要更新。
func Check() (CheckResult, error) {
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		return CheckResult{}, fmt.Errorf("获取最新版本失败: %w", err)
	}

	currentVersion, legacyVersion, err := fetchCurrentVersion()
	if err != nil {
		return CheckResult{}, fmt.Errorf("获取当前版本失败: %w", err)
	}

	result := CheckResult{
		LatestVersion:  latestVersion,
		CurrentVersion: currentVersion,
		LegacyVersion:  legacyVersion,
	}
	if legacyVersion {
		result.NeedUpdate = true
		return result, nil
	}

	result.NeedUpdate = compareVersion(currentVersion, latestVersion) < 0
	return result, nil
}

// CheckWithCurrentVersion 使用传入的当前版本号检查是否需要更新。
func CheckWithCurrentVersion(currentVersion string) (CheckResult, error) {
	latestVersion, err := fetchLatestVersion()
	if err != nil {
		return CheckResult{}, fmt.Errorf("获取最新版本失败: %w", err)
	}

	result := CheckResult{
		LatestVersion:  latestVersion,
		CurrentVersion: currentVersion,
	}
	result.NeedUpdate = compareVersion(currentVersion, latestVersion) < 0
	return result, nil
}

// fetchLatestVersion 从服务器上拉取最新版本号
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(latestVersionURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http请求失败，状态: %d, %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("版本响应为空")
	}
	if extractVersion(version) == "" {
		return "", fmt.Errorf("版本号格式错误: %q", version)
	}

	return version, nil
}

// 获取当前的版本号
func fetchCurrentVersion() (version string, legacyVersion bool, err error) {
	output, err := exec.Command(heyboxBotPath(), "--version").CombinedOutput()
	outputText := strings.TrimSpace(string(output))
	if err != nil {
		if strings.Contains(outputText, "flag provided but not defined") {
			return "", true, nil
		}
		return "", false, fmt.Errorf("%w: %s", err, outputText)
	}

	version = extractVersion(outputText)
	if version == "" {
		return "", false, fmt.Errorf("version not found in output: %q", outputText)
	}

	return version, false, nil
}

func compareVersion(current string, latest string) int {
	currentParts := versionParts(current)
	latestParts := versionParts(latest)
	length := max(len(latestParts), len(currentParts))

	for i := range length {
		currentPart := partAt(currentParts, i)
		latestPart := partAt(latestParts, i)
		if currentPart < latestPart {
			return -1
		}
		if currentPart > latestPart {
			return 1
		}
	}

	return 0
}

func heyboxBotPath() string {
	name := "heybox-bot"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	if currentExe, err := os.Executable(); err == nil {
		siblingPath := filepath.Join(filepath.Dir(currentExe), name)
		if _, err := os.Stat(siblingPath); err == nil {
			return siblingPath
		}
	}

	return name
}

func extractVersion(text string) string {
	return versionPattern.FindString(text)
}

func versionParts(version string) []int {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	rawParts := strings.Split(version, ".")
	parts := make([]int, 0, len(rawParts))
	for _, rawPart := range rawParts {
		part, err := strconv.Atoi(rawPart)
		if err != nil {
			part = 0
		}
		parts = append(parts, part)
	}
	return parts
}

func partAt(parts []int, index int) int {
	if index >= len(parts) {
		return 0
	}
	return parts[index]
}
