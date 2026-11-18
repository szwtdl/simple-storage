package service

import (
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/szwtdl/simple-storage/storage/common"
)

type LocalStorage struct {
	BasePath      string // 可信的文件根目录，不能由请求参数提供
	Domain        string
	MaxUploadSize int64 // 字节；0 使用默认的 100 MiB
}

// 使用 os.Root 在实际文件操作时约束路径，防止符号链接及检查后替换绕过。
func (l *LocalStorage) openRoot() (*os.Root, error) {
	return os.OpenRoot(l.BasePath)
}

func localName(name string, allowRoot bool) (string, error) {
	if allowRoot && name == "" {
		return ".", nil
	}
	if !filepath.IsLocal(name) || strings.Contains(name, `\`) {
		return "", errors.New("storage path must be relative and inside the storage root")
	}
	for _, part := range strings.Split(filepath.ToSlash(name), "/") {
		if part == ".." {
			return "", errors.New("parent directory references are not allowed")
		}
	}
	name = filepath.Clean(name)
	if name == "." && !allowRoot {
		return "", errors.New("file path is required")
	}
	return name, nil
}

func mkdirParents(root *os.Root, name string) error {
	dir := filepath.Dir(name)
	if dir == "." {
		return nil
	}
	current := ""
	for _, part := range strings.Split(dir, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		if err := root.Mkdir(current, 0750); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
			info, statErr := root.Stat(current)
			if statErr != nil {
				return statErr
			}
			if !info.IsDir() {
				return fmt.Errorf("storage parent is not a directory: %s", current)
			}
		}
	}
	return nil
}

func (l *LocalStorage) List(prefix string, size int) ([]common.File, error) {
	if err := validateListSize(size); err != nil {
		return nil, err
	}
	name, err := localName(prefix, true)
	if err != nil {
		return nil, err
	}
	root, err := l.openRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	files := make([]common.File, 0)
	var walk func(string) error
	walk = func(name string) error {
		info, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.Mode().IsRegular() {
			key := filepath.ToSlash(name)
			files = append(files, common.File{
				Name: info.Name(), Path: key, Size: info.Size(),
				Url:      strings.TrimRight(l.Domain, "/") + "/" + strings.ReplaceAll(url.PathEscape(key), "%2F", "/"),
				Modified: info.ModTime().Format(time.RFC3339),
			})
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		directory, err := root.Open(name)
		if err != nil {
			return err
		}
		defer directory.Close()
		// 按批读取，在达到 size 时立即停止，避免一次加载整个目录。
		for len(files) < size {
			entries, readErr := directory.ReadDir(128)
			for _, entry := range entries {
				if err := walk(filepath.Join(name, entry.Name())); err != nil {
					return err
				}
				if len(files) >= size {
					return nil
				}
			}
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			if readErr != nil {
				return readErr
			}
		}
		return nil
	}
	err = walk(name)
	return files, err
}

// writeFile 禁止覆盖并在复制失败时清理不完整文件。
func writeFile(root *os.Root, name string, reader io.Reader, limit int64) (bool, error) {
	if err := mkdirParents(root, name); err != nil {
		return false, err
	}
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return false, err
	}
	if limit > 0 {
		reader = io.LimitReader(reader, limit+1)
	}
	written, copyErr := io.Copy(file, reader)
	closeErr := file.Close()
	if limit > 0 && written > limit {
		copyErr = errors.New("file exceeds maximum upload size")
	}
	if err = errors.Join(copyErr, closeErr); err != nil {
		return false, errors.Join(err, root.Remove(name))
	}
	return true, nil
}

func (l *LocalStorage) Upload(file multipart.File, remote string) (bool, error) {
	if file == nil {
		return false, errors.New("upload file is required")
	}
	name, err := localName(remote, false)
	if err != nil {
		return false, err
	}
	root, err := l.openRoot()
	if err != nil {
		return false, err
	}
	defer root.Close()
	limit := l.MaxUploadSize
	if limit == 0 {
		limit = defaultMaxUploadSize
	}
	if limit < 0 || limit == math.MaxInt64 {
		return false, errors.New("invalid maximum upload size")
	}
	return writeFile(root, name, file, limit)
}

func openRegular(root *os.Root, name string) (*os.File, error) {
	info, err := root.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("source must be a regular file")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	info, err = file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("source must be a regular file")
	}
	return file, nil
}

func (l *LocalStorage) Download(remote string, local string) (bool, error) {
	name, err := localName(remote, false)
	if err != nil {
		return false, err
	}
	root, err := l.openRoot()
	if err != nil {
		return false, err
	}
	defer root.Close()
	src, err := openRegular(root, name)
	if err != nil {
		return false, err
	}
	defer src.Close()
	return saveDownload(local, src)
}

func (l *LocalStorage) Delete(remote string) (bool, error) {
	name, err := localName(remote, false)
	if err != nil {
		return false, err
	}
	root, err := l.openRoot()
	if err != nil {
		return false, err
	}
	defer root.Close()
	err = root.Remove(name)
	return err == nil, err
}

func (l *LocalStorage) Move(source string, target string) (bool, error) {
	srcName, err := localName(source, false)
	if err != nil {
		return false, err
	}
	dstName, err := localName(target, false)
	if err != nil {
		return false, err
	}
	root, err := l.openRoot()
	if err != nil {
		return false, err
	}
	defer root.Close()
	src, err := openRegular(root, srcName)
	if err != nil {
		return false, err
	}
	defer src.Close()
	// Go 1.24 的 Root 尚无 Rename；先独占创建并复制成功，再删除源文件。
	if _, err := writeFile(root, dstName, src, 0); err != nil {
		return false, err
	}
	if err := root.Remove(srcName); err != nil {
		return false, fmt.Errorf("target copied but source removal failed: %w", err)
	}
	return true, nil
}

func (l *LocalStorage) Generate(uploadDir string) (common.OSSPolicy, error) {
	dir, err := uploadPrefix(uploadDir)
	if err != nil {
		return common.OSSPolicy{}, err
	}
	return common.OSSPolicy{Host: l.Domain, Dir: dir, Expire: time.Now().Add(time.Hour).Unix()}, nil
}

func NewLocalStorage(config map[string]string) (*LocalStorage, error) {
	basePath := config["base_path"]
	if basePath == "" {
		basePath = "./uploads"
	}
	basePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}
	limit := int64(defaultMaxUploadSize)
	if value := config["max_upload_size"]; value != "" {
		limit, err = strconv.ParseInt(value, 10, 64)
		if err != nil || limit <= 0 || limit == math.MaxInt64 {
			return nil, errors.New("max_upload_size must be a positive byte count below MaxInt64")
		}
	}
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, err
	}
	return &LocalStorage{BasePath: basePath, Domain: config["base_url"], MaxUploadSize: limit}, nil
}
