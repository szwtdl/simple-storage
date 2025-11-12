package tests

import (
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/szwtdl/simple-storage/storage/service"
)

type ProviderFactory interface {
	NewAliYun(cfg map[string]string) (common.Storage, error)
	NewTencent(cfg map[string]string) (common.Storage, error)
	NewQiNiu(cfg map[string]string) (common.Storage, error)
}

// 默认工厂，用真实 service 函数

type RealFactory struct{}

func (f *RealFactory) NewAliYun(cfg map[string]string) (common.Storage, error) {
	return service.NewAliYun(cfg)
}
func (f *RealFactory) NewTencent(cfg map[string]string) (common.Storage, error) {
	return service.NewTencent(cfg)
}
func (f *RealFactory) NewQiNiu(cfg map[string]string) (common.Storage, error) {
	return service.NewQiNiu(cfg)
}
