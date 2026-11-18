# SimpleStorage

### 使用GO语言封装，阿里云Oss，腾讯云Cos，七牛云对象存储

### 安装

```bash
go get -u github.com/szwtdl/simple-storage
```

### 配置信息

```bash


    config := common.Config{
		Provider: "local",
		System: map[string]string{
			"base_path":     "",
			"base_url": "https://www.test.cn"
		},
	}

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

### 安全约定与兼容性变化

- 构建和部署要求 Go 1.26.11 或更新的受支持安全版本，避免旧标准库的已知漏洞。
- 云密钥通过环境变量或部署密钥管理系统提供，不写入代码或日志。本地 `main.go` 示例使用 `OSS_ENDPOINT`、`OSS_ACCESS_KEY_ID`、`OSS_ACCESS_KEY_SECRET`、`OSS_BUCKET`。曾经暴露的密钥必须在云控制台撤销或轮换，仅删除代码不能使旧密钥失效。
- `List(prefix, size)` 的 `size` 必须在 1–1000 之间，最多返回 `size` 个文件。
- 本地存储的远端路径必须位于 `base_path` 内，禁止绝对路径、`..` 和越界符号链接。`base_path` 及其上级目录必须由服务端配置并由可信用户管理，不应部署在可被其他租户修改的目录中。
- 本地上传默认限制为 100 MiB，可在配置中设置 `max_upload_size`（正整数，单位字节）。复制失败或超过限制会尝试删除不完整文件；新建目录权限为 0750，文件为 0600。
- 上传、移动和下载禁止直接覆盖现有目标。云存储依赖服务端的防覆盖能力；腾讯云在开启或暂停版本控制时会忽略防覆盖字段，需结合桶配置和权限策略处理。所有 `Download(remote, local)` 的 `local` 都是可信服务端导出路径，禁止直接透传客户端参数。
- 本地 `Move` 使用“独占创建并复制目标，再删除源文件”，不是原子重命名；如果删除源文件失败，会返回错误且保留已复制目标。云存储移动同样涉及复制和删除，应根据错误处理重试。
- `Generate(uploadDir)` 要求非空相对目录，返回的 `Dir` 统一以 `/` 结尾。七牛凭证绑定该目录前缀，并限制只新增和最大 100 MiB；腾讯直传上限修正为 100 MiB。阿里云直传维持原有 1 GiB 上限。
- 阿里云和腾讯直传时，前端必须将返回的 `form_fields` 中所有字段加入表单。阿里云还需提交原有 `OSSAccessKeyId`、`policy`、`signature`、`key` 等字段；腾讯签名字段已在 `form_fields` 中提供。`key` 应使用返回的 `Dir` 加服务端生成的 UUID 文件名，`file` 字段放在最后。不要将整份凭证写入日志。
- 阿里云 Endpoint 和七牛 Domain 强制使用 HTTPS，可省略协议；七牛下载拒绝跨域及 HTTP 降级重定向。
- 此项目是存储 SDK，没有 HTTP 接口或用户身份体系。接入层必须校验身份、对象所属租户和目录权限，服务端生成文件名，限制请求体大小及速率，按业务白名单校验扩展名和实际 MIME，拒绝可执行文件。通用对象存储不推断业务允许的文件类型；上传内容应放在不执行脚本的独立域名/目录，并配置适当的下载响应头。

安全实现参考：[Go 的路径越界防护](https://go.dev/blog/osroot)、[腾讯 COS 表单上传协议](https://cloud.tencent.com/document/product/436/14690)。


### 运行本地示例

直接执行 `go run .` 默认使用本地存储，目录为工作目录下的 `uploads`。可通过 `STORAGE_BASE_PATH` 和 `STORAGE_BASE_URL` 指定本地存储目录和访问域名。

使用阿里云时，在 GoLand 的 **Run → Edit Configurations → Environment variables** 中设置以下环境变量，再运行程序：

```text
STORAGE_PROVIDER=aliyun
OSS_ENDPOINT=https://oss-cn-qingdao.aliyuncs.com
OSS_ACCESS_KEY_ID=<你的 AccessKey ID>
OSS_ACCESS_KEY_SECRET=<你的 AccessKey Secret>
OSS_BUCKET=<你的存储桶名称>
```

也可在终端设置这些环境变量后运行 `go run .`。程序不会自动读取 `.env` 文件。`main.go` 被当前 `.gitignore` 忽略，作为本地运行示例使用。
