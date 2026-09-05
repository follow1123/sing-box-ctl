package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/uuid"
)

type Provider struct {
	workingDir string
}

// New 以工作目录构造 Provider。支持环境变量展开；相对路径基于当前进程目录解析，
// 结果归一化为绝对路径。
func New(workingDir string) (*Provider, error) {
	if workingDir == "" {
		return nil, fmt.Errorf("working dir is required")
	}
	dir := os.ExpandEnv(workingDir)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve working dir error:\n\t%w", err)
	}
	return &Provider{workingDir: filepath.Clean(abs)}, nil
}

// WorkingDir 返回解析后的工作目录（绝对路径）
func (p *Provider) WorkingDir() string {
	return p.workingDir
}

// ProvidersDir 返回 provider 数据目录
func (p *Provider) ProvidersDir() string {
	return filepath.Join(p.workingDir, "providers")
}

// ProviderDir 返回指定 provider 的数据目录
func (p *Provider) ProviderDir(uuid string) string {
	return filepath.Join(p.ProvidersDir(), uuid)
}

// SubscriptionDir 返回指定 provider 的订阅数据目录
func (p *Provider) SubscriptionDir(uuid string) string {
	return p.ProviderDir(uuid)
}

// TemplateDir 返回指定模板的数据目录
func (p *Provider) TemplateDir(uuid string) string {
	return filepath.Join(p.workingDir, "templates", uuid)
}

// SaveSubscription 保存订阅内容到 providers/<uuid>/，滚动保留 current/last/old 三份
func (p *Provider) SaveSubscription(uuid string, data []byte) error {
	if err := os.MkdirAll(p.SubscriptionDir(uuid), 0700); err != nil {
		return fmt.Errorf("create subscription dir error:\n\t%w", err)
	}
	return saveRolling(p.SubscriptionDir(uuid), data)
}

// ReadSubscription 读取指定 provider 的当前订阅内容
func (p *Provider) ReadSubscription(uuid string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(p.SubscriptionDir(uuid), "current"))
	if err != nil {
		return nil, fmt.Errorf("read subscription '%s' error:\n\t%w", uuid, err)
	}
	return data, nil
}

// SubscriptionVersions 返回订阅数据存在的历史版本（按 current/last/old 顺序）
func (p *Provider) SubscriptionVersions(uuid string) ([]string, error) {
	if !p.Exists(uuid) {
		return nil, fmt.Errorf("no provider with uuid: %s", uuid)
	}
	return versionsInDir(p.SubscriptionDir(uuid))
}

// RestoreSubscription 用指定历史版本覆盖当前订阅（会再次滚动保留一层历史）
func (p *Provider) RestoreSubscription(uuid, version string) error {
	if !p.Exists(uuid) {
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if !isValidVersion(version) {
		return fmt.Errorf("invalid version: %s", version)
	}
	data, err := readVersionFile(p.SubscriptionDir(uuid), version)
	if err != nil {
		return err
	}
	return p.SaveSubscription(uuid, data)
}

// saveRolling 滚动保存三份：old ← last ← current ← data
func saveRolling(dir string, data []byte) error {
	// old ← last
	lastFile := filepath.Join(dir, "last")
	if last, err := os.ReadFile(lastFile); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "old"), last, 0600); err != nil {
			return fmt.Errorf("save old file error:\n\t%w", err)
		}
	}
	// last ← current
	currentFile := filepath.Join(dir, "current")
	if current, err := os.ReadFile(currentFile); err == nil {
		if err := os.WriteFile(lastFile, current, 0600); err != nil {
			return fmt.Errorf("save last file error:\n\t%w", err)
		}
	}
	// current ← 新数据
	if err := os.WriteFile(currentFile, data, 0600); err != nil {
		return fmt.Errorf("save current file error:\n\t%w", err)
	}
	return nil
}

// 历史版本文件名白名单
const (
	versionCurrent = "current"
	versionLast    = "last"
	versionOld     = "old"
)

func isValidVersion(v string) bool {
	switch v {
	case versionCurrent, versionLast, versionOld:
		return true
	}
	return false
}

// versionsInDir 按 current/last/old 顺序列出目录中存在的版本文件
func versionsInDir(dir string) ([]string, error) {
	versions := make([]string, 0, 3)
	for _, v := range []string{versionCurrent, versionLast, versionOld} {
		if _, err := os.Stat(filepath.Join(dir, v)); err == nil {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

// readVersionFile 读取指定版本文件内容
func readVersionFile(dir, version string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(dir, version))
	if err != nil {
		return nil, fmt.Errorf("read version '%s' error:\n\t%w", version, err)
	}
	return data, nil
}

// ================= provider 元数据（文件系统实现） =================

// Add 新建 provider，返回 uuid。source 为 url 时写 url 文件；upload 时仅建目录，等待上传。
func (p *Provider) Add(name string, url string, source string, message string) (string, error) {
	if source == "" {
		source = SourceURL
	}
	if err := p.validate(name, url, source); err != nil {
		return "", err
	}
	// 重名检查
	if p.nameExists(name) {
		return "", fmt.Errorf("duplicate provider name '%s'", name)
	}

	uuid := uuid.NewString()
	dir := p.ProviderDir(uuid)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create provider dir error:\n\t%w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "name"), []byte(name), 0600); err != nil {
		return "", fmt.Errorf("write provider name error:\n\t%w", err)
	}
	if source == SourceURL {
		if err := os.WriteFile(filepath.Join(dir, "url"), []byte(url), 0600); err != nil {
			return "", fmt.Errorf("write provider url error:\n\t%w", err)
		}
	}
	if message != "" {
		if err := os.WriteFile(filepath.Join(dir, "message"), []byte(message), 0600); err != nil {
			return "", fmt.Errorf("write provider message error:\n\t%w", err)
		}
	}
	return uuid, nil
}

// Update 更新 provider 元数据。source 切为 url 时写 url 并清除 file_name；切为 upload 时清除 url。
func (p *Provider) Update(uuid string, name string, url string, source string, message string) error {
	if !p.Exists(uuid) {
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if source == "" {
		source = SourceURL
	}
	if err := p.validate(name, url, source); err != nil {
		return err
	}
	// 重名检查（排除自己）
	if other, _ := p.findByName(name); other != "" && other != uuid {
		return fmt.Errorf("duplicate provider name '%s'", name)
	}

	dir := p.ProviderDir(uuid)
	if err := os.WriteFile(filepath.Join(dir, "name"), []byte(name), 0600); err != nil {
		return fmt.Errorf("write provider name error:\n\t%w", err)
	}
	if source == SourceURL {
		if err := os.WriteFile(filepath.Join(dir, "url"), []byte(url), 0600); err != nil {
			return fmt.Errorf("write provider url error:\n\t%w", err)
		}
		// 切回 url 时清除上传文件名
		os.Remove(filepath.Join(dir, "file_name"))
	} else {
		os.Remove(filepath.Join(dir, "url"))
	}
	if message != "" {
		if err := os.WriteFile(filepath.Join(dir, "message"), []byte(message), 0600); err != nil {
			return fmt.Errorf("write provider message error:\n\t%w", err)
		}
	} else {
		os.Remove(filepath.Join(dir, "message"))
	}
	return nil
}

// SetFileName 记录上传文件名（upload 来源）
func (p *Provider) SetFileName(uuid string, fileName string) error {
	if !p.Exists(uuid) {
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if err := os.WriteFile(filepath.Join(p.ProviderDir(uuid), "file_name"), []byte(fileName), 0600); err != nil {
		return fmt.Errorf("write provider file_name error:\n\t%w", err)
	}
	return nil
}

// Delete 删除 provider（整个目录，含订阅数据）
func (p *Provider) Delete(uuid string) error {
	if !p.Exists(uuid) {
		return nil
	}
	if err := os.RemoveAll(p.ProviderDir(uuid)); err != nil {
		return fmt.Errorf("remove provider dir error:\n\t%w", err)
	}
	return nil
}

// Get 读取 provider 元数据；source 由文件推断（url 文件存在即有 url，否则为 upload）
func (p *Provider) Get(uuid string) *ProviderConfig {
	if !p.Exists(uuid) {
		return nil
	}
	dir := p.ProviderDir(uuid)
	info := &ProviderConfig{Uuid: uuid}

	name, _ := readFileContent(filepath.Join(dir, "name"))
	info.Name = name

	if urlContent, ok := readFileContent(filepath.Join(dir, "url")); ok && urlContent != "" {
		info.Source = SourceURL
		info.Url = urlContent
	} else {
		info.Source = SourceUpload
	}
	if fileName, ok := readFileContent(filepath.Join(dir, "file_name")); ok {
		info.FileName = fileName
	}
	if message, ok := readFileContent(filepath.Join(dir, "message")); ok {
		info.Message = message
	}
	return info
}

// List 列出所有 provider
func (p *Provider) List() []ProviderConfig {
	entries, err := os.ReadDir(p.ProvidersDir())
	if err != nil {
		return nil
	}
	list := make([]ProviderConfig, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if info := p.Get(e.Name()); info != nil {
			list = append(list, *info)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// Exists 判断 provider 是否存在
func (p *Provider) Exists(uuid string) bool {
	info, err := os.Stat(p.ProviderDir(uuid))
	return err == nil && info.IsDir()
}

// IsUploadSource 判断 provider 是否为上传来源（无 url 文件）
func (p *Provider) IsUploadSource(uuid string) bool {
	_, ok := readFileContent(filepath.Join(p.ProviderDir(uuid), "url"))
	return !ok
}

func (p *Provider) validate(name string, url string, source string) error {
	if source != SourceURL && source != SourceUpload {
		return fmt.Errorf("invalid source '%s', must be '%s' or '%s'", source, SourceURL, SourceUpload)
	}
	if source == SourceURL {
		if url == "" {
			return fmt.Errorf("url is required when source is '%s'", SourceURL)
		}
		if !isHTTPURL(url) {
			return fmt.Errorf("invalid url '%s', must be http(s) url", url)
		}
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (p *Provider) nameExists(name string) bool {
	uuid, _ := p.findByName(name)
	return uuid != ""
}

// findByName 按名称查找 provider uuid，不存在返回 ""
func (p *Provider) findByName(name string) (string, error) {
	entries, err := os.ReadDir(p.ProvidersDir())
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if content, ok := readFileContent(filepath.Join(p.ProviderDir(e.Name()), "name")); ok && content == name {
			return e.Name(), nil
		}
	}
	return "", nil
}

// readFileContent 读取文件内容；文件不存在时返回 false
func readFileContent(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
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
