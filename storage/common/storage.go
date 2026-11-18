package common

import "mime/multipart"

type File struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Url      string `json:"url"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type OSSPolicy struct {
	AccessKeyId string            `json:"OSSAccessKeyId"`
	Policy      string            `json:"policy"`
	Signature   string            `json:"signature"`
	Dir         string            `json:"dir"`
	Host        string            `json:"host"`
	Expire      int64             `json:"expire"`
	FormFields  map[string]string `json:"form_fields,omitempty"` // 直传时必须一并提交的表单字段
}

type Config struct {
	Provider string            // 云存储服务提供商: "aliyun", "tencent", "qiniu", "upyun"
	System   map[string]string // 配置信息
}

type Storage interface {
	List(prefix string, size int) ([]File, error)
	Upload(file multipart.File, remote string) (bool, error)
	Download(remote string, local string) (bool, error)
	Delete(remote string) (bool, error)
	Move(source string, target string) (bool, error)
	Generate(uploadDir string) (OSSPolicy, error)
}
