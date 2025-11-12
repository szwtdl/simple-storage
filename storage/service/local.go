package service

import (
	"errors"
	"github.com/szwtdl/simple-storage/storage/common"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalStorage struct {
	BasePath string // 文件根目录
	Domain   string // 外部访问域名
}

func (l *LocalStorage) List(prefix string, size int) ([]common.File, error) {
	var files []common.File
	root := filepath.Join(l.BasePath, prefix)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, common.File{
				Name:     info.Name(),
				Path:     strings.TrimPrefix(path, l.BasePath+"/"),
				Size:     info.Size(),
				Url:      l.Domain + "/" + strings.TrimPrefix(path, l.BasePath+"/"),
				Modified: info.ModTime().Format(time.RFC3339),
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	if len(files) > size {
		files = files[:size]
	}
	return files, nil
}

func (l *LocalStorage) Upload(file multipart.File, remote string) (bool, error) {
	dst := filepath.Join(l.BasePath, remote)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false, err
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		return false, err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, file)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (l *LocalStorage) Download(remote string, local string) (bool, error) {
	src := filepath.Join(l.BasePath, remote)
	if _, err := os.Stat(src); err != nil {
		return false, err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		return false, err
	}

	dstFile, err := os.Create(local)
	if err != nil {
		return false, err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err == nil, err
}

func (l *LocalStorage) Delete(remote string) (bool, error) {
	path := filepath.Join(l.BasePath, remote)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, errors.New("file not found")
	}
	return true, os.Remove(path)
}

func (l *LocalStorage) Move(source string, target string) (bool, error) {
	src := filepath.Join(l.BasePath, source)
	dst := filepath.Join(l.BasePath, target)

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false, err
	}
	return true, os.Rename(src, dst)
}

func (l *LocalStorage) Generate(uploadDir string) (common.OSSPolicy, error) {
	return common.OSSPolicy{
		Host:   l.Domain,
		Dir:    uploadDir,
		Expire: time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

func NewLocalStorage(config map[string]string) (*LocalStorage, error) {
	basePath := config["base_path"]
	if basePath == "" {
		basePath = "./uploads"
	}
	return &LocalStorage{
		BasePath: basePath,
		Domain:   config["base_url"],
	}, nil
}
