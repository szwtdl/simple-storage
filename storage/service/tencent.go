package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"net/url"
	"time"
)

type Tencent struct {
	client *cos.Client
	config map[string]string
}

func NewTencent(config map[string]string) (*Tencent, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", config["BucketName"], config["Endpoint"], config["Region"]))
	if err != nil {
		return nil, err
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config["AccessKey"],
			SecretKey: config["SecretKey"],
		},
	})
	return &Tencent{
		client: client,
		config: config,
	}, nil
}

func (s Tencent) List(prefix string, size int) ([]common.File, error) {
	var result []common.File
	opt := &cos.BucketGetOptions{
		Prefix:  prefix,
		MaxKeys: size,
	}
	v, _, err := s.client.Bucket.Get(context.Background(), opt)
	if err != nil {
		return nil, err
	}
	for _, obj := range v.Contents {
		result = append(result, common.File{
			Name: obj.Key,
			Size: obj.Size,
			Path: obj.Key,
		})
	}
	return result, nil
}

func (s Tencent) Upload(local string, remote string) (bool, error) {
	_, err := s.client.Object.PutFromFile(context.Background(), remote, local, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s Tencent) Download(remote string, local string) (bool, error) {
	_, err := s.client.Object.GetToFile(context.Background(), remote, local, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s Tencent) Delete(remote string) (bool, error) {
	_, err := s.client.Object.Delete(context.Background(), remote)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s Tencent) Move(source string, target string) (bool, error) {
	_, _, err := s.client.Object.Copy(context.Background(), target, source, nil)
	if err != nil {
		return false, err
	}
	_, err = s.client.Object.Delete(context.Background(), source)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s Tencent) Generate(uploadDir string) (common.OSSPolicy, error) {
	policy, signature, accessKeyId, expire, err := GenerateTencentPolicy(
		s.config["AccessKey"],
		s.config["SecretKey"],
		s.config["BucketName"],
		uploadDir,
		3600,
		100*1024*1024*1024, // 最大 100MB
	)
	if err != nil {
		return common.OSSPolicy{}, err
	}
	return common.OSSPolicy{
		AccessKeyId: accessKeyId,
		Policy:      policy,
		Signature:   signature,
		Dir:         uploadDir,
		Host:        fmt.Sprintf("%s.cos.%s.myqcloud.com", s.config["BucketName"], s.config["Region"]),
		Expire:      expire,
	}, nil
}

func GenerateTencentPolicy(secretID, secretKey, bucket, uploadDir string, expireSeconds int64, maxSize int64) (policyBase64, signature, accessKeyId string, expire int64, err error) {
	expire = time.Now().Unix() + expireSeconds
	expiration := time.Unix(expire, 0).UTC().Format("2006-01-02T15:04:05Z")
	policyStruct := map[string]interface{}{
		"expiration": expiration,
		"conditions": []interface{}{
			[]string{"starts-with", "$key", uploadDir},
			map[string]string{"bucket": bucket},
			[]interface{}{"content-length-range", 0, maxSize},
		},
	}
	policyJson, err := json.Marshal(policyStruct)
	if err != nil {
		return
	}
	policyBase64 = base64.StdEncoding.EncodeToString(policyJson)
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(policyBase64))
	signature = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	accessKeyId = secretID
	return
}
