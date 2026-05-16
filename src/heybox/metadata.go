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

func writeMetadataFile(path string, b []byte, perm os.FileMode) error {
	if err := os.MkdirAll(metadataDir, 0o700); err != nil {
		return fmt.Errorf("创建元数据目录失败: %w", err)
	}
	return os.WriteFile(path, b, perm)
}

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

func randomDeviceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 device_id 失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}

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

func formatMetadataTimestamp(timestamp float64) string {
	sec := int64(timestamp)
	nsec := int64((timestamp - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).Local().Format(metadataTimeFmt)
}

func parseMetadataTimestamp(timestamp string) (float64, error) {
	t, err := time.ParseInLocation(metadataTimeFmt, timestamp, time.Local)
	if err != nil {
		return 0, fmt.Errorf("解析元数据时间 %q 失败: %w", timestamp, err)
	}
	return float64(t.Unix()), nil
}

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
