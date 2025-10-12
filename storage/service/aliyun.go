package service

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/szwtdl/simple-storage/storage/common"
	"strings"
	"time"
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

func (s AliYun) Download(remote string, local string) (bool, error) {
	err := s.bucket.GetObjectToFile(remote, local)
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

func (s AliYun) Generate(uploadDir string) (common.OSSPolicy, error) {
	expireTime := time.Now().Add(10 * time.Minute).UTC()
	expireISO := expireTime.Format("2006-01-02T15:04:05.000Z")
	policy := map[string]interface{}{
		"expiration": expireISO,
		"conditions": []interface{}{
			map[string]string{"bucket": s.bucket.BucketName},
			[]interface{}{"starts-with", "$key", uploadDir},
			[]interface{}{"content-length-range", 0, 1 << 30}, // 最大 1GB
		},
	}
	// 转为 JSON，再 base64 编码
	policyJson, err := json.Marshal(policy)
	if err != nil {
		return common.OSSPolicy{}, fmt.Errorf("生成 policy JSON 失败: %v", err)
	}
	policyBase64 := base64.StdEncoding.EncodeToString(policyJson)
	h := hmac.New(sha1.New, []byte(s.client.Config.AccessKeySecret))
	h.Write([]byte(policyBase64))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return common.OSSPolicy{
		AccessKeyId: s.client.Config.AccessKeyID,
		Policy:      policyBase64,
		Signature:   signature,
		Dir:         strings.Trim(uploadDir, "/"),
		Host:        fmt.Sprintf("https://%s.%s", s.bucket.BucketName, s.client.Config.Endpoint),
		Expire:      expireTime.Unix(),
	}, nil
}
