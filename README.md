# SimpleStorage

### 使用GO语言封装，阿里云Oss，腾讯云Cos，七牛云对象存储

### 安装

```bash
go get -u github.com/szwtdl/simple-storage
```

### 配置信息

```bash

	config := common.Config{
		Provider: "aliyun",
		System: map[string]string{
			"Endpoint":   "https://oss-cn-qingdao.aliyuncs.com",
			"AccessKey":  "",
			"SecretKey":  "",
			"BucketName": "test",
			"Region":     "",
		},
	}
	
	config := common.Config{
		Provider: "tencent",
		System: map[string]string{
			"Endpoint":   "",
			"AccessKey":  "",
			"SecretKey":  "",
			"BucketName": "test",
			"Region":     "",
		},
	}

	config := common.Config{
		Provider: "qiniu",
		System: map[string]string{
			"Domain":     "",
			"AccessKey":  "",
			"SecretKey":  "",
			"BucketName": "test",
			"Region":     "",
		},
	}

```

### 初实话

```go
package main

import (
	"errors"
	"github.com/szwtdl/simple-storage/storage/common"
	"github.com/szwtdl/simple-storage/src/service"
	"github.com/szwtdl/simple-storage/storage/utils"
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

storage, err := NewStorage(config)
if err != nil {
fmt.Println(err)
return
}
// 获取当前目录
dir, err := os.Getwd()
if err != nil {
fmt.Println("Error:", err)
return
}
localFile := dir + "/temp/test.txt"
// 生产年月日目录
remoteFile := fmt.Sprintf("uploads/%s/%s", utils.GenerateDatePath(), utils.GenerateUUID()+".txt")
fmt.Println(localFile)
// 上传文件
if result, _ := storage.Upload(localFile, remoteFile); !result {
fmt.Println("上传失败")
}
fmt.Println("上传成功")
// 删除
if res, _ := storage.Delete("20240813/12313.txt"); !res {
fmt.Println("删除失败")
} else {
fmt.Println("删除成功")
}
// 列出文件
items, err := storage.List("", 20)
if err != nil {
fmt.Println(err)
return
}
for _, file := range items {
fmt.Println(file.Name)
}
```