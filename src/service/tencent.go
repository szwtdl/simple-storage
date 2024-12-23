package service

import (
	"context"
	"fmt"
	"github.com/szwtdl/simple-storage/src/common"
	"github.com/tencentyun/cos-go-sdk-v5"
	"net/http"
	"net/url"
)

type Tencent struct {
	client *cos.Client
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
	return &Tencent{client: client}, nil
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
