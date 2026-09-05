package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
)

// TemplateInfo 模板元信息（从文件系统读取）
type TemplateInfo struct {
	Uuid    string `json:"uuid"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
	// modTime 目录创建（近似）时间，仅用于排序，不参与 JSON 输出
	modTime time.Time
}

// TemplatesDir 返回模板目录
func (p *Provider) TemplatesDir() string {
	return filepath.Join(p.workingDir, "templates")
}

// AddTemplate 新建模板，返回模板 uuid。初始内容由调用方通过 SaveTemplate 写入。
func (p *Provider) AddTemplate(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if p.templateNameExists(name) {
		return "", fmt.Errorf("duplicate template name '%s'", name)
	}
	uuid := uuid.NewString()
	dir := p.TemplateDir(uuid)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create template dir error:\n\t%w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "name"), []byte(name), 0600); err != nil {
		return "", fmt.Errorf("write template name error:\n\t%w", err)
	}
	return uuid, nil
}

// DeleteTemplate 删除模板（整个目录）
func (p *Provider) DeleteTemplate(uuid string) error {
	if !p.TemplateExists(uuid) {
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	if err := os.RemoveAll(p.TemplateDir(uuid)); err != nil {
		return fmt.Errorf("remove template dir error:\n\t%w", err)
	}
	return nil
}

// ListTemplates 列出所有模板（uuid 目录扫描）
func (p *Provider) ListTemplates() ([]TemplateInfo, error) {
	entries, err := os.ReadDir(p.TemplatesDir())
	if err != nil {
		return nil, fmt.Errorf("read templates dir error:\n\t%w", err)
	}
	infos := make([]TemplateInfo, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		uuid := e.Name()
		name, err := p.readTemplateName(uuid)
		if err != nil {
			name = uuid // name 文件缺失时用 uuid 兜底
		}
		// 排序时间取模板本体 name 文件（创建时写入一次、之后不变）的 mtime
		modTime := time.Time{}
		if info, err := os.Stat(filepath.Join(p.TemplateDir(uuid), "name")); err == nil {
			modTime = info.ModTime()
		}
		infos = append(infos, TemplateInfo{
			Uuid:    uuid,
			Name:    name,
			Default: p.IsDefaultTemplate(uuid),
			modTime: modTime,
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		// 默认模板排最前（固定在最左），其余按创建时间倒序（最新创建的在前）
		if infos[i].Default != infos[j].Default {
			return infos[i].Default
		}
		if !infos[i].modTime.Equal(infos[j].modTime) {
			return infos[i].modTime.After(infos[j].modTime)
		}
		return infos[i].Uuid < infos[j].Uuid
	})
	return infos, nil
}

// templateNameExists 判断是否存在同名模板
func (p *Provider) templateNameExists(name string) bool {
	entries, err := os.ReadDir(p.TemplatesDir())
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if content, ok := readFileContent(filepath.Join(p.TemplateDir(e.Name()), "name")); ok && content == name {
			return true
		}
	}
	return false
}

// TemplateExists 判断模板是否存在
func (p *Provider) TemplateExists(uuid string) bool {
	info, err := os.Stat(p.TemplateDir(uuid))
	return err == nil && info.IsDir()
}

// GetTemplate 获取模板元信息，不存在返回 nil
func (p *Provider) GetTemplate(uuid string) *TemplateInfo {
	if !p.TemplateExists(uuid) {
		return nil
	}
	name, err := p.readTemplateName(uuid)
	if err != nil {
		name = uuid
	}
	return &TemplateInfo{Uuid: uuid, Name: name, Default: p.IsDefaultTemplate(uuid)}
}

// IsDefaultTemplate 判断模板是否为默认（存在 default 空文件）
func (p *Provider) IsDefaultTemplate(uuid string) bool {
	_, err := os.Stat(filepath.Join(p.TemplateDir(uuid), "default"))
	return err == nil
}

// SetDefaultTemplate 将指定模板设为默认，并清除其他模板的默认标记
func (p *Provider) SetDefaultTemplate(uuid string) error {
	if !p.TemplateExists(uuid) {
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	infos, err := p.ListTemplates()
	if err != nil {
		return err
	}
	for _, info := range infos {
		defaultFile := filepath.Join(p.TemplateDir(info.Uuid), "default")
		if info.Uuid == uuid {
			if _, err := os.Stat(defaultFile); err != nil {
				if err := os.WriteFile(defaultFile, nil, 0600); err != nil {
					return fmt.Errorf("set default template error:\n\t%w", err)
				}
			}
		} else {
			os.Remove(defaultFile)
		}
	}
	return nil
}

// DefaultTemplate 返回默认模板 uuid；没有 default 标记时取第一个模板兜底
func (p *Provider) DefaultTemplate() (string, error) {
	infos, err := p.ListTemplates()
	if err != nil {
		return "", err
	}
	if len(infos) == 0 {
		return "", fmt.Errorf("no templates")
	}
	for _, info := range infos {
		if info.Default {
			return info.Uuid, nil
		}
	}
	return infos[0].Uuid, nil
}

// SaveTemplate 保存模板内容到 templates/<uuid>/，滚动保留 current/last/old
func (p *Provider) SaveTemplate(uuid string, data []byte) error {
	if !p.TemplateExists(uuid) {
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	return saveRolling(p.TemplateDir(uuid), data)
}

// ReadTemplate 读取模板当前内容
func (p *Provider) ReadTemplate(uuid string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(p.TemplateDir(uuid), "current"))
	if err != nil {
		return nil, fmt.Errorf("read template '%s' error:\n\t%w", uuid, err)
	}
	return data, nil
}

func (p *Provider) readTemplateName(uuid string) (string, error) {
	data, err := os.ReadFile(filepath.Join(p.TemplateDir(uuid), "name"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
