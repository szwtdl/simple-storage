package service

import (
	"context"
	"fmt"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/szwtdl/simple-storage/storage/common"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type QiNiu struct {
	bucket   string
	uploader *storage.FormUploader
	manager  *storage.BucketManager
	mac      *qbox.Mac // 保存 mac
	domain   string
}

func NewQiNiu(config map[string]string) (*QiNiu, error) {
	mac := qbox.NewMac(config["AccessKey"], config["SecretKey"])
	cfg := storage.Config{
		Zone:          &storage.ZoneHuadong,
		UseHTTPS:      true,
		UseCdnDomains: true,
	}
	manager := storage.NewBucketManager(mac, &cfg)
	uploader := storage.NewFormUploader(&cfg)
	return &QiNiu{
		bucket:   config["BucketName"],
		uploader: uploader,
		manager:  manager,
		mac:      mac, // 保存
		domain:   config["Domain"],
	}, nil
}

func (s QiNiu) List(prefix string, size int) ([]common.File, error) {
	var files []common.File
	delimiter := "" // 用于分隔目录
	marker := ""    // 初始标记为空
	for {
		entries, _, nextMarker, hasNext, err := s.manager.ListFiles(s.bucket, prefix, delimiter, marker, size)
		if err != nil {
			return nil, err
		}
		// 添加文件到结果列表
		for _, entry := range entries {
			files = append(files, common.File{
				Name: entry.Key,
				Size: entry.Fsize,
				Path: entry.Key,
			})
		}

		// 如果没有更多文件，结束循环
		if !hasNext {
			break
		}

		// 更新标记，继续下一页
		marker = nextMarker
	}

	return files, nil
}

func (s QiNiu) Upload(file multipart.File, filename string) (bool, error) {
	putPolicy := storage.PutPolicy{
		Scope: s.bucket,
	}
	upToken := putPolicy.UploadToken(s.mac)
	ret := storage.PutRet{}
	ctx := context.Background()
	err := s.uploader.Put(ctx, &ret, upToken, filename, file, -1, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s QiNiu) Download(remote string, local string) (bool, error) {
	expires := time.Now().Add(time.Hour).Unix()
	privateUrl := storage.MakePrivateURL(s.mac, s.domain, remote, expires)
	resp, err := http.Get(privateUrl)
	if err != nil {
		return false, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("qiniu download failed, http status: %s", resp.Status)
	}
	file, err := os.Create(local)
	if err != nil {
		return false, fmt.Errorf("create file error: %w", err)
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return false, fmt.Errorf("write file error: %w", err)
	}
	return true, nil
}

func (s QiNiu) Delete(remote string) (bool, error) {
	err := s.manager.Delete(s.bucket, remote)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s QiNiu) Move(source string, target string) (bool, error) {
	err := s.manager.Move(s.bucket, source, s.bucket, target, true)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s QiNiu) Generate(uploadDir string) (common.OSSPolicy, error) {
	putPolicy := storage.PutPolicy{
		Scope: s.bucket,
	}
	upToken := putPolicy.UploadToken(s.mac)
	expire := time.Now().Add(time.Hour).Unix()
	return common.OSSPolicy{
		AccessKeyId: s.manager.Mac.AccessKey,
		Policy:      upToken,
		Signature:   "", // 七牛token没有单独signature，留空或自行处理
		Dir:         uploadDir,
		Host:        s.domain,
		Expire:      expire,
	}, nil
}
