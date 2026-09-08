package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/follow1123/sing-box-ctl/throttle"
	"github.com/google/uuid"
)

// 模板保存相关常量：变更只提交节流任务，由统一回调延迟落盘
const (
	templateSaveKey   = "templates:save"
	templateSaveDelay = 10 * time.Second
)

// TemplateManagerData 是 templates/data.json 的结构：模板元数据 + 内容版本槽位
type TemplateManagerData struct {
	// DefaultTemplate 默认模板 uuid
	DefaultTemplate string          `json:"default_template"`
	Templates       []TemplateEntry `json:"templates"`
}

// TemplateEntry 一个用户模板的完整数据（内置种子不落盘，不在其中）
type TemplateEntry struct {
	Uuid string `json:"uuid"`
	Name string `json:"name"`
	// CreatedAt 创建时间：用于列表排序（最新创建在前），迁移时取旧 name 文件 mtime
	CreatedAt time.Time `json:"created_at"`
	// Nodes 内容版本槽位，最多 3 份：下标 0 = 当前，1 = 上次，2 = 最旧
	// Data 为模板 JSON 原文（与编辑器往返一致，无格式转换）
	Nodes []*SubscriptionNode `json:"nodes,omitempty"`
}

// TemplateInfo 模板元信息（对外输出，webui/前端契约）
type TemplateInfo struct {
	Uuid    string `json:"uuid"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

// TemplateManager 内存态管理用户模板数据，统一保存到 templates/data.json
type TemplateManager struct {
	workingDir string
	path       string
	throttle   *throttle.Throttler

	mu   sync.Mutex // 保护 data 与文件保存
	data *TemplateManagerData
}

// NewTemplateManager 以工作目录构造 TemplateManager；加载 data.json
// （损坏/缺失回退 .bak），都没有时若有历史散目录数据则一次性迁移。
func NewTemplateManager(workingDir string) (*TemplateManager, error) {
	if workingDir == "" {
		return nil, fmt.Errorf("working dir is required")
	}
	dir := os.ExpandEnv(workingDir)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve working dir error:\n\t%w", err)
	}
	abs = filepath.Clean(abs)
	tm := &TemplateManager{
		workingDir: abs,
		path:       filepath.Join(abs, "templates", "data.json"),
		throttle:   throttle.New(),
		data:       &TemplateManagerData{Templates: make([]TemplateEntry, 0)},
	}
	if err := tm.load(); err != nil {
		return nil, err
	}
	return tm, nil
}

func (tm *TemplateManager) WorkingDir() string {
	return tm.workingDir
}

// Path 返回 data.json 的完整路径
func (tm *TemplateManager) Path() string {
	return tm.path
}

// load 优先读 data.json；文件缺失或损坏时回退 .bak；都没有则尝试迁移历史散目录
func (tm *TemplateManager) load() error {
	dir := filepath.Dir(tm.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create templates dir error:\n\t%w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	data, err := os.ReadFile(tm.path)
	if err == nil {
		if err := json.Unmarshal(data, tm.data); err != nil {
			return fmt.Errorf("parse %s error:\n\t%w", tm.path, err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s error:\n\t%w", tm.path, err)
	}

	bakData, bakErr := os.ReadFile(tm.path + ".bak")
	if bakErr == nil {
		if err := json.Unmarshal(bakData, tm.data); err != nil {
			return fmt.Errorf("parse %s error:\n\t%w", tm.path+".bak", err)
		}
		return nil
	}
	if !errors.Is(bakErr, os.ErrNotExist) {
		return fmt.Errorf("read %s error:\n\t%w", tm.path+".bak", bakErr)
	}

	// 全新的工作目录：看是否有历史散目录数据可迁移
	if migrated, err := tm.migrateLegacy(); err != nil {
		return err
	} else if migrated {
		return tm.save()
	}
	return nil
}

// save 统一保存：先把当前 data.json 备份为 .bak，再原子写入新文件
func (tm *TemplateManager) save() error {
	data, err := json.MarshalIndent(tm.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal template data error:\n\t%w", err)
	}

	tmp := tm.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write template data tmp error:\n\t%w", err)
	}
	if _, err := os.Stat(tm.path); err == nil {
		if err := os.Rename(tm.path, tm.path+".bak"); err != nil {
			return fmt.Errorf("backup template data error:\n\t%w", err)
		}
	}
	if err := os.Rename(tmp, tm.path); err != nil {
		return fmt.Errorf("save template data error:\n\t%w", err)
	}
	return nil
}

// SaveNow 立即把当前内存态落盘
//
// TODO(退出 flush): 服务停止 / 进程退出路径应调用本方法，避免最后窗口内的变更丢失
func (tm *TemplateManager) SaveNow() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.save()
}

// scheduleSave 提交延迟保存任务（节流合并多次变更，落盘即当时完整内存态）
func (tm *TemplateManager) scheduleSave() {
	tm.throttle.Submit(templateSaveKey, templateSaveDelay, func() {
		tm.mu.Lock()
		err := tm.save()
		tm.mu.Unlock()
		if err != nil {
			// TODO(保存失败日志): 延迟落盘失败无人感知，需要留痕
		}
	})
}

// compactJSON 把模板内容归一为紧凑单行（存储用，保证跨落盘/重载字节稳定）
func compactJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// prettyJSON 输出 2 空格缩进的规范文本（返回给编辑器用，幂等稳定）
func prettyJSON(raw []byte) ([]byte, error) {
	compact, err := compactJSON(raw)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, compact, "", "  "); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// migrateLegacy 把历史散目录（templates/<uuid>/ 的 name/current/last/old/default）
// 一次性导入 data.json，迁移完成后旧目录保留。返回是否有数据可迁移。调用方需持锁。
func (tm *TemplateManager) migrateLegacy() (bool, error) {
	entries, err := os.ReadDir(tm.TemplatesDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read templates dir error:\n\t%w", err)
	}

	migrated := false
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		uuid := e.Name()
		if len(uuid) < 2 || uuid == "data" {
			continue
		}
		dir := filepath.Join(tm.TemplatesDir(), uuid)
		entry := TemplateEntry{Uuid: uuid}

		if content, ok := readFileContent(filepath.Join(dir, "name")); ok {
			entry.Name = content
		}
		if info, err := os.Stat(filepath.Join(dir, "name")); err == nil {
			entry.CreatedAt = info.ModTime()
		}
		// 内容版本：current/last/old -> 下标 0/1/2（模板为 JSON 原文，无需转换）
		for _, v := range []string{versionKeyCurrent, versionKeyLast, versionKeyOld} {
			raw, err := os.ReadFile(filepath.Join(dir, v))
			if err != nil {
				continue
			}
			compact, err := compactJSON(raw)
			if err != nil {
				continue
			}
			entry.Nodes = append(entry.Nodes, &SubscriptionNode{Data: compact})
		}
		tm.data.Templates = append(tm.data.Templates, entry)
		// default 空文件标记
		if _, err := os.Stat(filepath.Join(dir, "default")); err == nil && tm.data.DefaultTemplate == "" {
			tm.data.DefaultTemplate = uuid
		}
		migrated = true
	}
	return migrated, nil
}

func (tm *TemplateManager) TemplatesDir() string {
	return filepath.Join(tm.workingDir, "templates")
}

// ================= 模板元数据 =================

// AddTemplate 新建模板，返回模板 uuid。内容由调用方通过 SaveTemplate 写入。
func (tm *TemplateManager) AddTemplate(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	tm.mu.Lock()
	for i := range tm.data.Templates {
		if tm.data.Templates[i].Name == name {
			tm.mu.Unlock()
			return "", fmt.Errorf("duplicate template name '%s'", name)
		}
	}
	entry := TemplateEntry{
		Uuid:      uuid.NewString(),
		Name:      name,
		CreatedAt: time.Now(),
	}
	tm.data.Templates = append(tm.data.Templates, entry)
	tm.mu.Unlock()
	tm.scheduleSave()
	return entry.Uuid, nil
}

// DeleteTemplate 删除模板。默认模板不可删（由 webui 层校验提示）；删除默认会清空标记
func (tm *TemplateManager) DeleteTemplate(uuid string) error {
	tm.mu.Lock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		tm.mu.Unlock()
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	if tm.data.DefaultTemplate == uuid {
		tm.data.DefaultTemplate = ""
	}
	tm.data.Templates = append(tm.data.Templates[:idx], tm.data.Templates[idx+1:]...)
	tm.mu.Unlock()
	tm.scheduleSave()
	return nil
}

// ListTemplates 列出所有模板：默认模板最前，其余按创建时间倒序（最新创建在前）
func (tm *TemplateManager) ListTemplates() []TemplateInfo {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	infos := make([]TemplateInfo, 0, len(tm.data.Templates))
	for _, e := range tm.data.Templates {
		infos = append(infos, TemplateInfo{
			Uuid:    e.Uuid,
			Name:    e.Name,
			Default: e.Uuid == tm.data.DefaultTemplate,
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Default != infos[j].Default {
			return infos[i].Default
		}
		e1, e2 := tm.findEntryLocked(infos[i].Uuid), tm.findEntryLocked(infos[j].Uuid)
		if !e1.CreatedAt.Equal(e2.CreatedAt) {
			return e1.CreatedAt.After(e2.CreatedAt)
		}
		return infos[i].Uuid < infos[j].Uuid
	})
	return infos
}

// GetTemplate 获取模板元信息；不存在返回 nil
func (tm *TemplateManager) GetTemplate(uuid string) *TemplateInfo {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		return nil
	}
	return &TemplateInfo{
		Uuid:    uuid,
		Name:    tm.data.Templates[idx].Name,
		Default: uuid == tm.data.DefaultTemplate,
	}
}

// TemplateExists 判断模板是否存在
func (tm *TemplateManager) TemplateExists(uuid string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.indexLocked(uuid) >= 0
}

// SetDefaultTemplate 将指定模板设为默认（唯一）
func (tm *TemplateManager) SetDefaultTemplate(uuid string) error {
	tm.mu.Lock()
	if tm.indexLocked(uuid) < 0 {
		tm.mu.Unlock()
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	tm.data.DefaultTemplate = uuid
	tm.mu.Unlock()
	tm.scheduleSave()
	return nil
}

// DefaultTemplate 返回默认模板 uuid；没有默认标记时取第一个模板兜底
func (tm *TemplateManager) DefaultTemplate() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.data.Templates) == 0 {
		return "", fmt.Errorf("no templates")
	}
	if tm.indexLocked(tm.data.DefaultTemplate) >= 0 {
		return tm.data.DefaultTemplate, nil
	}
	// 兜底：取列表第一个（默认最前，其次最新创建）
	first := tm.data.Templates[0]
	for i := 1; i < len(tm.data.Templates); i++ {
		if tm.data.Templates[i].CreatedAt.After(first.CreatedAt) {
			first = tm.data.Templates[i]
		}
	}
	return first.Uuid, nil
}

// ================= 模板内容版本 =================

// SaveTemplate 保存新一版模板 JSON（滚动保留最多 3 份，内容 compact 存储）
func (tm *TemplateManager) SaveTemplate(uuid string, data []byte) error {
	tm.mu.Lock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		tm.mu.Unlock()
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	compact, err := compactJSON(data)
	if err != nil {
		tm.mu.Unlock()
		return fmt.Errorf("invalid json content")
	}
	entry := &tm.data.Templates[idx]
	nodes := []*SubscriptionNode{{Data: compact}}
	for _, n := range entry.Nodes {
		if len(nodes) >= 3 {
			break
		}
		nodes = append(nodes, n)
	}
	entry.Nodes = nodes
	tm.mu.Unlock()
	tm.scheduleSave()
	return nil
}

// ReadTemplate 读取模板当前内容（2 空格缩进规范文本，供编辑器直接使用）
func (tm *TemplateManager) ReadTemplate(uuid string) ([]byte, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		return nil, fmt.Errorf("no template with uuid: %s", uuid)
	}
	nodes := tm.data.Templates[idx].Nodes
	if len(nodes) == 0 {
		return nil, fmt.Errorf("template '%s' has no content", uuid)
	}
	return prettyJSON(nodes[0].Data)
}

// TemplateVersions 返回模板存在的历史版本（current/last/old 顺序）
func (tm *TemplateManager) TemplateVersions(uuid string) ([]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		return nil, fmt.Errorf("no template with uuid: %s", uuid)
	}
	count := len(tm.data.Templates[idx].Nodes)
	versions := make([]string, 0, count)
	for _, v := range []string{versionKeyCurrent, versionKeyLast, versionKeyOld} {
		if versionKeyIndex[v] < count {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

// RestoreTemplate 用指定历史版本覆盖当前内容（会再次滚动保留一层历史）
func (tm *TemplateManager) RestoreTemplate(uuid, version string) error {
	tm.mu.Lock()
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		tm.mu.Unlock()
		return fmt.Errorf("no template with uuid: %s", uuid)
	}
	pos, ok := versionKeyIndex[version]
	if !ok {
		tm.mu.Unlock()
		return fmt.Errorf("invalid version: %s", version)
	}
	nodes := tm.data.Templates[idx].Nodes
	if pos >= len(nodes) {
		tm.mu.Unlock()
		return fmt.Errorf("version '%s' not exists", version)
	}
	data := append([]byte(nil), nodes[pos].Data...)
	tm.mu.Unlock()
	return tm.SaveTemplate(uuid, data)
}

// 以下 *Locked 方法假定调用方已持有 tm.mu
func (tm *TemplateManager) indexLocked(uuid string) int {
	for i := range tm.data.Templates {
		if tm.data.Templates[i].Uuid == uuid {
			return i
		}
	}
	return -1
}

func (tm *TemplateManager) findEntryLocked(uuid string) *TemplateEntry {
	idx := tm.indexLocked(uuid)
	if idx < 0 {
		return nil
	}
	return &tm.data.Templates[idx]
}
