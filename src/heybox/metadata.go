package heybox

import (
	"fmt"
	"os"
)

const (
	metadataDir            = "metadata"
	sessionFile            = metadataDir + "/session.json"
	legacySessionFile      = "session.json"
	messageStateFile       = metadataDir + "/message_state.json"
	legacyMessageStateFile = "message_state.json"
)

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
