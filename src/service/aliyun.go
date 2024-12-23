package service

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/szwtdl/simple-storage/src/common"
)

type AliYun struct {
	client *oss.Client
	bucket *oss.Bucket
}

func NewAliYun(config map[string]string) (*AliYun, error) {
	client, err := oss.New(config["Endpoint"], config["AccessKey"], config["SecretKey"])
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(config["BucketName"])
	if err != nil {
		return nil, err
	}
	return &AliYun{client: client, bucket: bucket}, nil
}

func (s AliYun) List(prefix string, size int) ([]common.File, error) {
	marker := ""
	var result []common.File
	options := []oss.Option{
		oss.Prefix(prefix),
		oss.MaxKeys(size),
	}
	for {
		lsRes, err := s.bucket.ListObjects(append(options, oss.Marker(marker))...)
		if err != nil {
			return nil, err
		}
		for _, obj := range lsRes.Objects {
			result = append(result, common.File{
				Name: obj.Key,
				Size: obj.Size,
				Path: obj.Key,
			})
		}
		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}
	return result, nil
}

func (s AliYun) Upload(local string, remote string) (bool, error) {
	err := s.bucket.PutObjectFromFile(remote, local)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s AliYun) Delete(remote string) (bool, error) {
	err := s.bucket.DeleteObject(remote)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s AliYun) Move(source string, target string) (bool, error) {
	_, err := s.bucket.CopyObject(target, source)
	if err != nil {
		return false, err
	}
	err = s.bucket.DeleteObject(source)
	if err != nil {
		return false, err
	}
	return true, nil
}
