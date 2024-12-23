package service

import (
	"context"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/szwtdl/simple-storage/src/common"
)

type QiNiu struct {
	bucket   string
	uploader *storage.FormUploader
	manager  *storage.BucketManager
	domain   string
}

func NewQiNiu(config map[string]string) (*QiNiu, error) {
	mac := qbox.NewMac(config["AccessKey"], config["SecretKey"])
	cfg := storage.Config{
		Zone:          &storage.ZoneHuadong, // 根据存储区域选择
		UseHTTPS:      true,
		UseCdnDomains: true,
	}
	manager := storage.NewBucketManager(mac, &cfg)
	uploader := storage.NewFormUploader(&cfg)
	return &QiNiu{
		bucket:   config["BucketName"],
		uploader: uploader,
		manager:  manager,
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

func (s QiNiu) Upload(local string, remote string) (bool, error) {
	putPolicy := storage.PutPolicy{
		Scope: s.bucket,
	}
	mac := qbox.NewMac(s.manager.Mac.AccessKey, string(s.manager.Mac.SecretKey))
	upToken := putPolicy.UploadToken(mac)
	ret := storage.PutRet{}
	ctx := context.Background() // Create a non-nil context
	err := s.uploader.PutFile(ctx, &ret, upToken, remote, local, nil)
	if err != nil {
		return false, err
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
