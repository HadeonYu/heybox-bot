package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Update 拉取当前平台的最新 release 文件并覆盖安装。
func Update() error {
	return update()
}

func update() error {
	info, err := fetch()
	if err != nil {
		return err
	}

	platform := releasePlatform()
	file, ok := info.Files[platform]
	if !ok {
		return fmt.Errorf("没有当前平台的发行版: %s", platform)
	}

	tempDir, err := os.MkdirTemp("", "heybox-bot-update-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "release"+archiveSuffix(file.URL))
	if err := downloadFile(file.URL, archivePath); err != nil {
		return err
	}
	if err := verifySHA256(archivePath, file.SHA256); err != nil {
		return err
	}

	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0o700); err != nil {
		return fmt.Errorf("创建解压目录失败: %w", err)
	}
	if err := extractArchive(archivePath, extractDir); err != nil {
		return err
	}

	sourceDir, err := packageContentDir(extractDir)
	if err != nil {
		return err
	}
	if err := removeSkippedFiles(sourceDir); err != nil {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前可执行文件路径失败: %w", err)
	}
	return copyDir(sourceDir, filepath.Dir(exePath))
}

func extractArchive(archivePath string, targetDir string) error {
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, targetDir)
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGZ(archivePath, targetDir)
	default:
		return fmt.Errorf("不支持的发行版压缩格式: %s", archivePath)
	}
}

func extractZip(archivePath string, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开 zip 失败: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath, err := safeJoin(targetDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		source, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开 zip 文件内容失败: %w", err)
		}
		err = writeExtractedFile(targetPath, source, file.Mode())
		source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGZ(archivePath string, targetDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开 tar.gz 失败: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("读取 gzip 失败: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 tar 失败: %w", err)
		}

		targetPath, err := safeJoin(targetDir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			if err := writeExtractedFile(targetPath, tarReader, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeExtractedFile(path string, reader io.Reader, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("创建解压文件失败: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("写入解压文件失败: %w", err)
	}
	return nil
}

func packageContentDir(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("读取解压目录失败: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}
	return extractDir, nil
}

func removeSkippedFiles(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if shouldSkipReleaseFile(entry.Name()) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("删除 %s 失败: %w", path, err)
			}
		}
		return nil
	})
}

func shouldSkipReleaseFile(name string) bool {
	switch name {
	case "config.yaml", "system_prompt.md", updaterBinaryName():
		return true
	default:
		return false
	}
}

func copyDir(sourceDir string, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(sourcePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if sourcePath == sourceDir {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}
		targetPath := filepath.Join(targetDir, relPath)

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("读取文件信息失败: %w", err)
		}
		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		return copyFile(sourcePath, targetPath, info.Mode())
	})
}

func copyFile(sourcePath string, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}
	return nil
}

func safeJoin(root string, name string) (string, error) {
	cleanName := filepath.Clean(name)
	if cleanName == "." || filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return "", fmt.Errorf("压缩包包含非法路径: %s", name)
	}

	targetPath := filepath.Join(root, cleanName)
	relPath, err := filepath.Rel(root, targetPath)
	if err != nil {
		return "", fmt.Errorf("检查解压路径失败: %w", err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("压缩包包含非法路径: %s", name)
	}
	return targetPath, nil
}

func archiveSuffix(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	path := rawURL
	if err == nil {
		path = parsedURL.Path
	}

	switch {
	case strings.HasSuffix(path, ".tar.gz"):
		return ".tar.gz"
	case strings.HasSuffix(path, ".tgz"):
		return ".tgz"
	case strings.HasSuffix(path, ".zip"):
		return ".zip"
	default:
		return filepath.Ext(path)
	}
}

func updaterBinaryName() string {
	if runtime.GOOS == "windows" {
		return "heybox-bot-update.exe"
	}
	return "heybox-bot-update"
}

func releasePlatform() string {
	return releaseOS() + "_" + runtime.GOARCH
}

func releaseOS() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}
