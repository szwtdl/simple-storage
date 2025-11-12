package storage

import (
	"errors"
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/szwtdl/simple-storage/storage/service"
	"strings"
)

func NewStorage(config common.Config) (common.Storage, error) {
	provider := strings.TrimSpace(strings.ToLower(config.Provider))
	if provider == "" {
		return nil, errors.New("storage provider not specified")
	}
	switch provider {
	case "aliyun":
		return service.NewAliYun(config.System)
	case "tencent":
		return service.NewTencent(config.System)
	case "qiniu":
		return service.NewQiNiu(config.System)
	case "local":
		return service.NewLocalStorage(config.System)
	default:
		return nil, errors.New("unsupported storage provider: " + provider)
	}
}
