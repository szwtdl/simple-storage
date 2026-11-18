package tests

import (
	"errors"
	"github.com/szwtdl/simple-storage/storage/common"
	"mime/multipart"
	"testing"
)

// mockStorage 实现 common.Storage 接口
type mockStorage struct{}

func (m *mockStorage) List(prefix string, size int) ([]common.File, error) {
	// 返回空列表
	return []common.File{}, nil
}

func (m *mockStorage) Upload(file multipart.File, remote string) (bool, error) {
	return true, nil
}

func (m *mockStorage) Download(remote string, local string) (bool, error) {
	return true, nil
}

func (m *mockStorage) Delete(remote string) (bool, error) {
	return true, nil
}

func (m *mockStorage) Move(source string, target string) (bool, error) {
	return true, nil
}

func (m *mockStorage) Generate(uploadDir string) (common.OSSPolicy, error) {
	return common.OSSPolicy{
		AccessKeyId: "mock",
		Policy:      "mock",
		Signature:   "mock",
		Dir:         "mock",
		Host:        "mock",
		Expire:      0,
	}, nil
}

type mockFactory struct{}

func (m *mockFactory) NewAliYun(cfg map[string]string) (common.Storage, error) {
	return &mockStorage{}, nil
}
func (m *mockFactory) NewTencent(cfg map[string]string) (common.Storage, error) {
	return &mockStorage{}, nil
}
func (m *mockFactory) NewQiNiu(cfg map[string]string) (common.Storage, error) {
	return &mockStorage{}, nil
}

func NewStorageWithFactory(config common.Config, factory ProviderFactory) (common.Storage, error) {
	switch config.Provider {
	case "aliyun":
		return factory.NewAliYun(config.System)
	case "tencent":
		return factory.NewTencent(config.System)
	case "qiniu":
		return factory.NewQiNiu(config.System)
	default:
		return nil, errors.New("unsupported storage provider")
	}
}

func TestNewStorage(t *testing.T) {
	mock := &mockFactory{}

	tests := []struct {
		name      string
		provider  string
		expectErr bool
	}{
		{"AliYun", "aliyun", false},
		{"Tencent", "tencent", false},
		{"QiNiu", "qiniu", false},
		{"Unsupported", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := common.Config{Provider: tt.provider}

			s, err := NewStorageWithFactory(cfg, mock)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if s == nil {
					t.Errorf("expected non-nil storage instance")
				}
			}
		})
	}
}
