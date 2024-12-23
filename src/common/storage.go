package common

type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type Config struct {
	Provider string            // 云存储服务提供商: "aliyun", "tencent", "qiniu", "upyun"
	System   map[string]string // 配置信息
}

type Storage interface {
	List(prefix string, size int) ([]File, error)
	Upload(local string, remote string) (bool, error)
	Delete(remote string) (bool, error)
	Move(source string, target string) (bool, error)
}
