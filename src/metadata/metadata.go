package metadata

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"heybox-bot/heybox/api"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	metadataDir       = "metadata"
	SessionFile       = metadataDir + "/session.json"
	LegacySessionFile = "session.json"
	metadataFile      = metadataDir + "/metadata.json"
	metadataTimeFmt   = time.DateTime
	frequencyWindow   = time.Minute
)

var (
	currentMetadata *metadata
	metadataMu      sync.Mutex
)

type metadata struct {
	LastAtMessageTime string             `json:"last_at_message_time"`
	UserCallTimes     map[string][]int64 `json:"user_call_times"`
	UserCallCounts    map[string]int     `json:"user_call_counts,omitempty"`
	DeviceID          string             `json:"device_id"`
	XHHTokenID        string             `json:"x_xhh_tokenid"`
}

// Init 加载或初始化全局元数据并设置 API 设备 ID。
func Init() error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	md, err := loadMetadataLocked()
	if err != nil {
		return fmt.Errorf("初始化元数据失败: %w", err)
	}
	currentMetadata = md
	api.SetDeviceID(md.DeviceID)
	api.SetXHHTokenID(md.XHHTokenID)
	return nil
}

// ReadFile 读取元数据文件并在需要时回退读取旧路径。
func ReadFile(path, legacyPath string) ([]byte, bool, error) {
	b, err := os.ReadFile(path)
	if err == nil || !os.IsNotExist(err) || legacyPath == "" {
		return b, false, err
	}

	b, err = os.ReadFile(legacyPath)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

// WriteFile 创建元数据目录并写入指定文件内容。
func WriteFile(path string, b []byte, perm os.FileMode) error {
	if err := os.MkdirAll(metadataDir, 0o700); err != nil {
		return fmt.Errorf("创建元数据目录失败: %w", err)
	}
	return os.WriteFile(path, b, perm)
}

// loadMetadataLocked 在持锁状态下读取或创建 metadata.json。
func loadMetadataLocked() (*metadata, error) {
	b, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			md := newDefaultMetadata()
			return md, saveMetadataLocked(md)
		}
		return nil, fmt.Errorf("读取元数据失败: %w", err)
	}

	md := &metadata{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, md); err != nil {
			return nil, fmt.Errorf("解析元数据失败: %w", err)
		}
	}

	changed := false
	if md.LastAtMessageTime == "" {
		md.LastAtMessageTime = formatMetadataTimestamp(float64(time.Now().Unix()))
		changed = true
	}
	if md.UserCallTimes == nil {
		md.UserCallTimes = make(map[string][]int64)
		changed = true
	}
	if len(md.UserCallCounts) > 0 {
		now := time.Now().Unix()
		for userID, count := range md.UserCallCounts {
			for range count {
				md.UserCallTimes[userID] = append(md.UserCallTimes[userID], now)
			}
		}
		md.UserCallCounts = nil
		changed = true
	}
	if md.DeviceID == "" {
		deviceID, err := randomDeviceID()
		if err != nil {
			return nil, err
		}
		md.DeviceID = deviceID
		changed = true
	}
	if md.XHHTokenID == "" {
		tokenID, err := newXHHTokenID()
		if err != nil {
			return nil, err
		}
		md.XHHTokenID = tokenID
		changed = true
	}
	if changed {
		if err := saveMetadataLocked(md); err != nil {
			return nil, err
		}
	}

	return md, nil
}

// newDefaultMetadata 创建带当前时间和随机设备 ID 的默认元数据。
func newDefaultMetadata() *metadata {
	deviceID, err := randomDeviceID()
	if err != nil {
		deviceID = fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return &metadata{
		LastAtMessageTime: formatMetadataTimestamp(float64(time.Now().Unix())),
		UserCallTimes:     make(map[string][]int64),
		DeviceID:          deviceID,
		XHHTokenID:        mustNewXHHTokenID(),
	}
}

// randomDeviceID 生成 32 位十六进制设备 ID。
func randomDeviceID() (string, error) {
	return randomHex32("生成 device_id 失败")
}

// randomHex32 生成 32 位十六进制随机字符串。
func randomHex32(errMsg string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	return hex.EncodeToString(b), nil
}

// mustNewXHHTokenID 生成 x_xhh_tokenid 并在失败时使用时间兜底。
func mustNewXHHTokenID() string {
	tokenID, err := newXHHTokenID()
	if err == nil {
		return tokenID
	}
	return fallbackXHHTokenID()
}

// newXHHTokenID 按小黑盒网页端算法生成 x_xhh_tokenid。
func newXHHTokenID() (string, error) {
	randomText1, err := randomHex32("生成 x_xhh_tokenid 随机文本失败1")
	if err != nil {
		return "", err
	}
	randomText2, err := randomHex32("生成 x_xhh_tokenid 随机文本失败2")
	if err != nil {
		return "", err
	}
	randomText3, err := randomHex32("生成 x_xhh_tokenid 随机文本失败3")
	if err != nil {
		return "", err
	}

	return encodeXHHTokenID(
		strconv.Itoa(int(time.Now().Unix())),
		randomText1,
		randomText2,
		randomText3,
	), nil
}

// fallbackXHHTokenID 使用当前时间构造兜底的 x_xhh_tokenid。
func fallbackXHHTokenID() string {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	return encodeXHHTokenID(now, now+"1", now+"2", now+"3")
}

// encodeXHHTokenID 将时间文本和随机文本混合编码为 x_xhh_tokenid。
func encodeXHHTokenID(parts ...string) string {
	raw := make([]byte, 0, md5.Size*len(parts)+1)
	for _, part := range parts {
		sum := md5.Sum([]byte(part))
		raw = append(raw, sum[:]...)
	}
	raw = append(raw, 0)
	return base64.StdEncoding.EncodeToString(raw)
}

// saveMetadataLocked 在持锁状态下将元数据保存到文件。
func saveMetadataLocked(md *metadata) error {
	b, err := json.MarshalIndent(md, "", "\t")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}
	if err := WriteFile(metadataFile, b, 0o600); err != nil {
		return fmt.Errorf("写入元数据失败: %w", err)
	}
	return nil
}

// formatMetadataTimestamp 将浮点秒数时间戳格式化为可读时间。
func formatMetadataTimestamp(timestamp float64) string {
	sec := int64(timestamp)
	nsec := int64((timestamp - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).Local().Format(metadataTimeFmt)
}

// parseMetadataTimestamp 将元数据中的可读时间解析为浮点秒数。
func parseMetadataTimestamp(timestamp string) (float64, error) {
	t, err := time.ParseInLocation(metadataTimeFmt, timestamp, time.Local)
	if err != nil {
		return 0, fmt.Errorf("解析元数据时间 %q 失败: %w", timestamp, err)
	}
	return float64(t.Unix()), nil
}

// LoadLastAtMessageTimestamp 读取上次处理 @ 消息的时间戳。
func LoadLastAtMessageTimestamp() (float64, error) {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	if currentMetadata == nil {
		md, err := loadMetadataLocked()
		if err != nil {
			return 0, err
		}
		currentMetadata = md
		api.SetDeviceID(md.DeviceID)
		api.SetXHHTokenID(md.XHHTokenID)
	}
	return parseMetadataTimestamp(currentMetadata.LastAtMessageTime)
}

// SaveLastAtMessageTimestamp 保存上次处理 @ 消息的时间戳。
func SaveLastAtMessageTimestamp(timestamp float64) error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	if currentMetadata == nil {
		md, err := loadMetadataLocked()
		if err != nil {
			return err
		}
		currentMetadata = md
	}
	currentMetadata.LastAtMessageTime = formatMetadataTimestamp(timestamp)
	if err := saveMetadataLocked(currentMetadata); err != nil {
		return err
	}
	api.SetDeviceID(currentMetadata.DeviceID)
	api.SetXHHTokenID(currentMetadata.XHHTokenID)
	return nil
}

// SaveLastAtMessageTime 保存上次成功处理 @ 消息的时间。
func SaveLastAtMessageTime(timestamp time.Time) error {
	return SaveLastAtMessageTimestamp(float64(timestamp.UnixNano()) / 1e9)
}

// RecordUserCall 记录指定用户调用时间，并返回最近一分钟内的调用次数。
func RecordUserCall(userID string) (int, error) {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	if currentMetadata == nil {
		md, err := loadMetadataLocked()
		if err != nil {
			return 0, err
		}
		currentMetadata = md
	}
	if currentMetadata.UserCallTimes == nil {
		currentMetadata.UserCallTimes = make(map[string][]int64)
	}

	now := time.Now().Unix()
	calls := pruneExpiredCallTimes(currentMetadata.UserCallTimes[userID], now)
	calls = append(calls, now)
	currentMetadata.UserCallTimes[userID] = calls
	if err := saveMetadataLocked(currentMetadata); err != nil {
		return 0, err
	}
	return len(calls), nil
}

// CleanupUserCallTimes 清理所有用户过期调用时间，列表为空后删除用户项。
func CleanupUserCallTimes() error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	if currentMetadata == nil {
		md, err := loadMetadataLocked()
		if err != nil {
			return err
		}
		currentMetadata = md
	}
	if len(currentMetadata.UserCallTimes) == 0 {
		return nil
	}

	now := time.Now().Unix()
	for userID, calls := range currentMetadata.UserCallTimes {
		calls = pruneExpiredCallTimes(calls, now)
		if len(calls) == 0 {
			delete(currentMetadata.UserCallTimes, userID)
			continue
		}
		currentMetadata.UserCallTimes[userID] = calls
	}
	return saveMetadataLocked(currentMetadata)
}

func pruneExpiredCallTimes(calls []int64, now int64) []int64 {
	cutoff := now - int64(frequencyWindow.Seconds())
	start := 0
	for start < len(calls) && calls[start] <= cutoff {
		start++
	}
	return calls[start:]
}
