package service

import (
	"errors"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const defaultMaxUploadSize = 100 * 1024 * 1024

func validateListSize(size int) error {
	if size < 1 || size > 1000 {
		return errors.New("list size must be between 1 and 1000")
	}
	return nil
}

// 目录凭证必须以 / 结尾，避免 uploads/user1 同时授权 uploads/user10。
func uploadPrefix(dir string) (string, error) {
	if dir == "" || strings.ContainsAny(dir, `\:`) || strings.HasPrefix(dir, "/") {
		return "", errors.New("a non-empty relative upload directory is required")
	}
	for _, part := range strings.Split(dir, "/") {
		if part == ".." || part == "." {
			return "", errors.New("invalid upload directory")
		}
	}
	dir = path.Clean(dir)
	if dir == "." {
		return "", errors.New("invalid upload directory")
	}
	return dir + "/", nil
}

func httpsEndpoint(value string) (string, error) {
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	endpoint, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil ||
		(endpoint.Path != "" && endpoint.Path != "/") || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return "", errors.New("storage endpoint must be an HTTPS origin without credentials")
	}
	return strings.TrimRight(endpoint.String(), "/"), nil
}

// local 是调用方指定的可信导出路径，不接受 HTTP 请求中的原始路径。
func saveDownload(local string, reader io.Reader) (bool, error) {
	if local == "" {
		return false, errors.New("download destination is required")
	}
	destination, err := filepath.Abs(local)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
		return false, err
	}
	root, err := os.OpenRoot(filepath.Dir(destination))
	if err != nil {
		return false, err
	}
	defer root.Close()
	return writeFile(root, filepath.Base(destination), reader, 0)
}
