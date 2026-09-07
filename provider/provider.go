package provider

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// Provider 承载模板的文件系统存储与工作目录信息；provider 自身数据
// （元数据 + 订阅节点版本）由 ProviderManager 管理。
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

// ProvidersDir 返回 provider 数据目录（ProviderManager 的 data.json 所在目录）
func (p *Provider) ProvidersDir() string {
	return filepath.Join(p.workingDir, "providers")
}

// TemplateDir 返回指定模板的数据目录
func (p *Provider) TemplateDir(uuid string) string {
	return filepath.Join(p.workingDir, "templates", uuid)
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

// 历史版本文件名白名单（模板文件系统用）
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
