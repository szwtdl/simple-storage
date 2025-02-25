package storage

import (
	"errors"
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/szwtdl/simple-storage/storage/service"
)

func NewStorage(config common.Config) (common.Storage, error) {
	switch config.Provider {
	case "aliyun":
		return service.NewAliYun(config.System)
	case "tencent":
		return service.NewTencent(config.System)
	case "qiniu":
		return service.NewQiNiu(config.System)
	default:
		return nil, errors.New("unsupported storage provider")
	}
}
