package provider

const (
	// SourceURL 通过 url 下载订阅
	SourceURL = "url"
	// SourceUpload 通过 webui 上传配置文件
	SourceUpload = "upload"
)

// SingBoxCtlConfig 是主配置文件结构（-c 指定）
type SingBoxCtlConfig struct {
	WorkingDir string           `json:"working_dir"`
	Providers  []ProviderConfig `json:"providers"`
}

type ProviderConfig struct {
	Uuid    string `json:"uuid"`
	Name    string `json:"name"`
	Url     string `json:"url"`
	Source  string `json:"source"`
	Message string `json:"message,omitempty"`
}
