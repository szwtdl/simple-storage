package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/tencentyun/cos-go-sdk-v5"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
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
		Timeout: 5 * time.Minute,
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
	if err := validateListSize(size); err != nil {
		return nil, err
	}
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

func (s Tencent) Upload(file multipart.File, remote string) (bool, error) {
	headers := http.Header{"X-Cos-Forbid-Overwrite": []string{"true"}}
	_, err := s.client.Object.Put(context.Background(), remote, file, &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{XOptionHeader: &headers},
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s Tencent) Download(remote string, local string) (bool, error) {
	response, err := s.client.Object.Get(context.Background(), remote, nil)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	_, err = saveDownload(local, response.Body)
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
	headers := http.Header{"X-Cos-Forbid-Overwrite": []string{"true"}}
	// COS 要求复制源包含当前存储桶的 host，避免将对象 key 解释为其他桶。
	copySource := s.client.BaseURL.BucketURL.Host + "/" + source
	_, _, err := s.client.Object.Copy(context.Background(), target, copySource, &cos.ObjectCopyOptions{
		ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{XOptionHeader: &headers},
	})
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
	dir, err := uploadPrefix(uploadDir)
	if err != nil {
		return common.OSSPolicy{}, err
	}
	policy, signature, accessKeyId, expire, err := GenerateTencentPolicy(
		s.config["AccessKey"],
		s.config["SecretKey"],
		strings.Split(s.client.BaseURL.BucketURL.Host, ".")[0],
		dir,
		3600,
		defaultMaxUploadSize, // 最大 100MB
	)
	if err != nil {
		return common.OSSPolicy{}, err
	}
	fields := map[string]string{
		"q-sign-algorithm": "sha1", "q-ak": accessKeyId, "q-signature": signature,
		"q-key-time":             fmt.Sprintf("%d;%d", expire-3600, expire),
		"x-cos-forbid-overwrite": "true", "policy": policy,
	}
	return common.OSSPolicy{
		FormFields:  fields,
		AccessKeyId: accessKeyId,
		Policy:      policy,
		Signature:   signature,
		Dir:         dir,
		Host:        s.client.BaseURL.BucketURL.String(),
		Expire:      expire,
	}, nil
}

func GenerateTencentPolicy(secretID, secretKey, bucket, uploadDir string, expireSeconds int64, maxSize int64) (policyBase64, signature, accessKeyId string, expire int64, err error) {
	uploadDir, err = uploadPrefix(uploadDir)
	if err != nil {
		return
	}
	if expireSeconds <= 0 || expireSeconds > 3600 || maxSize <= 0 || maxSize > defaultMaxUploadSize {
		err = fmt.Errorf("policy expiry must be 1..3600 seconds and size 1..100 MiB")
		return
	}
	start := time.Now().Unix()
	expire = start + expireSeconds
	keyTime := fmt.Sprintf("%d;%d", start, expire)
	expiration := time.Unix(expire, 0).UTC().Format("2006-01-02T15:04:05Z")
	policyStruct := map[string]interface{}{
		"expiration": expiration,
		"conditions": []interface{}{
			[]string{"starts-with", "$key", uploadDir},
			map[string]string{"bucket": bucket},
			map[string]string{"q-sign-algorithm": "sha1"},
			map[string]string{"q-ak": secretID},
			map[string]string{"q-sign-time": keyTime},
			map[string]string{"x-cos-forbid-overwrite": "true"},
			[]interface{}{"content-length-range", 0, maxSize},
		},
	}
	policyJson, err := json.Marshal(policyStruct)
	if err != nil {
		return
	}
	policyBase64 = base64.StdEncoding.EncodeToString(policyJson)
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(keyTime))
	signKey := hex.EncodeToString(mac.Sum(nil))
	policyHash := sha1.Sum(policyJson)
	mac = hmac.New(sha1.New, []byte(signKey))
	mac.Write([]byte(hex.EncodeToString(policyHash[:])))
	signature = hex.EncodeToString(mac.Sum(nil))
	accessKeyId = secretID
	return
}
