package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"
)

type Provider struct {
	path   string
	config *SingBoxCtlConfig
}

func New(path string) (*Provider, error) {
	config := &SingBoxCtlConfig{}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s error:\n\t%w", path, err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("unmarshal json error:\n\t%w", err)
		}
	}

	// 解析 working_dir：支持环境变量和相对路径（基于配置文件所在目录）
	workingDir := os.ExpandEnv(config.WorkingDir)
	if !filepath.IsAbs(workingDir) {
		workingDir = filepath.Join(filepath.Dir(path), workingDir)
	}
	config.WorkingDir = filepath.Clean(workingDir)

	return &Provider{
		path:   path,
		config: config,
	}, nil
}

// WorkingDir 返回解析后的工作目录（绝对路径）
func (p *Provider) WorkingDir() string {
	return p.config.WorkingDir
}

// SubscriptionDir 返回指定 provider 的订阅数据目录
func (p *Provider) SubscriptionDir(uuid string) string {
	return filepath.Join(p.config.WorkingDir, "providers", uuid)
}

// SaveSubscription 保存订阅内容到 providers/<uuid>/，滚动保留 current/last/old 三份
func (p *Provider) SaveSubscription(uuid string, data []byte) error {
	dir := p.SubscriptionDir(uuid)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create subscription dir error:\n\t%w", err)
	}

	// old ← last
	lastFile := filepath.Join(dir, "last")
	if last, err := os.ReadFile(lastFile); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "old"), last, 0600); err != nil {
			return fmt.Errorf("save old subscription error:\n\t%w", err)
		}
	}
	// last ← current
	currentFile := filepath.Join(dir, "current")
	if current, err := os.ReadFile(currentFile); err == nil {
		if err := os.WriteFile(lastFile, current, 0600); err != nil {
			return fmt.Errorf("save last subscription error:\n\t%w", err)
		}
	}
	// current ← 新数据
	if err := os.WriteFile(currentFile, data, 0600); err != nil {
		return fmt.Errorf("save current subscription error:\n\t%w", err)
	}
	return nil
}

// ReadSubscription 读取指定 provider 的当前订阅内容
func (p *Provider) ReadSubscription(uuid string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(p.SubscriptionDir(uuid), "current"))
	if err != nil {
		return nil, fmt.Errorf("read subscription '%s' error:\n\t%w", uuid, err)
	}
	return data, nil
}

func (p *Provider) Add(name string, url string, source string, message string) error {
	if source == "" {
		source = SourceURL
	}
	if !isValidSource(source) {
		return fmt.Errorf("invalid source '%s', must be '%s' or '%s'", source, SourceURL, SourceUpload)
	}
	if source == SourceURL && url == "" {
		return fmt.Errorf("url is required when source is '%s'", SourceURL)
	}
	if source == SourceURL && !isHTTPURL(url) {
		return fmt.Errorf("invalid url '%s', must be http(s) url", url)
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}

	for _, prov := range p.config.Providers {
		if prov.Name == name {
			return fmt.Errorf("duplicate provider name '%s'", name)
		}
	}

	p.config.Providers = append(p.config.Providers, ProviderConfig{
		Uuid:    uuid.NewString(),
		Name:    name,
		Url:     url,
		Source:  source,
		Message: message,
	})
	return nil
}

func (p *Provider) Update(uuid string, name string, url string, source string, message string) error {
	idx := p.indexOf(uuid)
	if idx < 0 {
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if source == "" {
		source = SourceURL
	}
	if !isValidSource(source) {
		return fmt.Errorf("invalid source '%s', must be '%s' or '%s'", source, SourceURL, SourceUpload)
	}
	if source == SourceURL && url == "" {
		return fmt.Errorf("url is required when source is '%s'", SourceURL)
	}
	if source == SourceURL && !isHTTPURL(url) {
		return fmt.Errorf("invalid url '%s', must be http(s) url", url)
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}

	for i, prov := range p.config.Providers {
		if i != idx && prov.Name == name {
			return fmt.Errorf("duplicate provider name '%s'", name)
		}
	}

	p.config.Providers[idx].Name = name
	p.config.Providers[idx].Url = url
	p.config.Providers[idx].Source = source
	p.config.Providers[idx].Message = message
	return nil
}

func (p *Provider) Delete(uuid string) error {
	idx := p.indexOf(uuid)
	if idx < 0 {
		return nil
	}
	p.config.Providers = slices.Delete(p.config.Providers, idx, idx+1)

	// 清理该 provider 的订阅数据目录
	if err := os.RemoveAll(p.SubscriptionDir(uuid)); err != nil {
		return fmt.Errorf("remove subscription dir error:\n\t%w", err)
	}
	return nil
}

func (p *Provider) Get(uuid string) *ProviderConfig {
	for i := range p.config.Providers {
		if p.config.Providers[i].Uuid == uuid {
			return &p.config.Providers[i]
		}
	}
	return nil
}

func (p *Provider) List() []ProviderConfig {
	return p.config.Providers
}

func (p *Provider) Save() error {
	data, err := json.MarshalIndent(p.config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal to json error:\n\t%w", err)
	}

	if err := os.WriteFile(p.path, data, 0660); err != nil {
		return fmt.Errorf("save config to %s error:\n\t%w", p.path, err)
	}
	return nil
}

func (p *Provider) indexOf(uuid string) int {
	for i, prov := range p.config.Providers {
		if prov.Uuid == uuid {
			return i
		}
	}
	return -1
}

func isValidSource(source string) bool {
	return source == SourceURL || source == SourceUpload
}

// DataFromSource 从 url 或本地文件读取数据
func DataFromSource(source string) ([]byte, error) {
	var data bytes.Buffer
	if isHTTPURL(source) {
		resp, err := http.Get(source)
		if err != nil {
			return nil, fmt.Errorf("failed to download from URL:\n\t%w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response from server: %s", resp.Status)
		}

		_, err = io.Copy(&data, resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body:\n\t%w", err)
		}
	} else {
		absPath, err := filepath.Abs(source)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve file path:\n\t%w", err)
		}
		d, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file:\n\t%w", err)
		}
		_, err = data.Write(d)
		if err != nil {
			return nil, fmt.Errorf("save data to buffer error:\n\t%w", err)
		}
	}
	return data.Bytes(), nil
}

// 判断是否是 HTTP/HTTPS URL
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
