package heybox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"heybox-bot/heybox/api"
	"os"
	"sync"
	"time"
)

const (
	metadataDir       = "metadata"
	sessionFile       = metadataDir + "/session.json"
	legacySessionFile = "session.json"
	metadataFile      = metadataDir + "/metadata.json"
	metadataTimeFmt   = time.DateTime
)

var (
	currentMetadata *metadata
	metadataMu      sync.Mutex
)

type metadata struct {
	LastAtMessageTime string `json:"last_at_message_time"`
	DeviceID          string `json:"device_id"`
}

// readMetadataFile 读取元数据文件并在需要时回退读取旧路径。
func readMetadataFile(path, legacyPath string) ([]byte, bool, error) {
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

// writeMetadataFile 创建元数据目录并写入指定文件内容。
func writeMetadataFile(path string, b []byte, perm os.FileMode) error {
	if err := os.MkdirAll(metadataDir, 0o700); err != nil {
		return fmt.Errorf("创建元数据目录失败: %w", err)
	}
	return os.WriteFile(path, b, perm)
}

// initMetadata 加载或初始化全局元数据并设置 API 设备 ID。
func initMetadata() error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	md, err := loadMetadataLocked()
	if err != nil {
		return err
	}
	currentMetadata = md
	api.SetDeviceID(md.DeviceID)
	return nil
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
	if md.DeviceID == "" {
		deviceID, err := randomDeviceID()
		if err != nil {
			return nil, err
		}
		md.DeviceID = deviceID
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
		DeviceID:          deviceID,
	}
}

// randomDeviceID 生成 32 位十六进制设备 ID。
func randomDeviceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 device_id 失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// saveMetadataLocked 在持锁状态下将元数据保存到文件。
func saveMetadataLocked(md *metadata) error {
	b, err := json.MarshalIndent(md, "", "\t")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}
	if err := writeMetadataFile(metadataFile, b, 0o600); err != nil {
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

// loadLastAtMessageTimestamp 读取上次处理 @ 消息的时间戳。
func loadLastAtMessageTimestamp() (float64, error) {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	if currentMetadata == nil {
		md, err := loadMetadataLocked()
		if err != nil {
			return 0, err
		}
		currentMetadata = md
		api.SetDeviceID(md.DeviceID)
	}
	return parseMetadataTimestamp(currentMetadata.LastAtMessageTime)
}

// saveLastAtMessageTimestamp 保存上次处理 @ 消息的时间戳。
func saveLastAtMessageTimestamp(timestamp float64) error {
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
	return nil
}
