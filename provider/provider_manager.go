package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/throttle"
	"github.com/google/uuid"
)

// 保存相关常量：所有变更只提交节流任务，由统一回调延迟落盘
const (
	// saveKey 节流 key：manager 内写死，所有变更共用
	saveKey = "providers:save"
	// saveDelay 变更后延迟多久统一保存一次
	saveDelay = 10 * time.Second
)

// ManagerData 是 providers/data.json 的结构：provider 元数据 + 节点版本槽位
type ManagerData struct {
	// DefaultProvider 默认 provider uuid（语义待前端阶段定义，先支持存取）
	DefaultProvider string          `json:"default_provider"`
	Providers       []ProviderEntry `json:"providers"`
}

// ProviderEntry 一个 provider 的完整数据（元数据 + 节点历史）
type ProviderEntry struct {
	ProviderConfig
	// Nodes 节点版本槽位，最多 3 份：下标 0 = 当前，1 = 上次，2 = 最旧
	Nodes []*SubscriptionNode `json:"nodes,omitempty"`
}

// SubscriptionNode 一个版本槽位；Data 是具体 JSON 内容（sing-box outbounds 数组）
type SubscriptionNode struct {
	Data json.RawMessage `json:"data"`
}

// 版本槽位 key（对外语义与 webui versions 接口一致）
const (
	subVersionCurrent = "current"
	subVersionLast    = "last"
	subVersionOld     = "old"
)

var subVersionIndex = map[string]int{subVersionCurrent: 0, subVersionLast: 1, subVersionOld: 2}

// ProviderManager 内存态管理 provider 数据；变更只改内存，由节流任务统一延迟落盘
type ProviderManager struct {
	workingDir string
	path       string
	throttle   *throttle.Throttler

	mu   sync.Mutex // 保护 data 与文件保存（webui 并发请求 + 定时保存回调）
	data *ManagerData
}

// NewManager 以工作目录构造 ProviderManager；加载 data.json（损坏/缺失回退 .bak），
// 都没有时若有历史散目录数据则一次性迁移。
func NewManager(workingDir string) (*ProviderManager, error) {
	if workingDir == "" {
		return nil, fmt.Errorf("working dir is required")
	}
	dir := os.ExpandEnv(workingDir)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve working dir error:\n\t%w", err)
	}
	abs = filepath.Clean(abs)
	pm := &ProviderManager{
		workingDir: abs,
		path:       filepath.Join(abs, "providers", "data.json"),
		throttle:   throttle.New(),
		data:       &ManagerData{Providers: make([]ProviderEntry, 0)},
	}
	if err := pm.load(); err != nil {
		return nil, err
	}
	return pm, nil
}

func (pm *ProviderManager) WorkingDir() string {
	return pm.workingDir
}

// Path 返回 data.json 的完整路径
func (pm *ProviderManager) Path() string {
	return pm.path
}

// load 优先读 data.json；文件缺失或损坏时回退 .bak；都没有则尝试迁移历史散目录
func (pm *ProviderManager) load() error {
	dir := filepath.Dir(pm.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create providers dir error:\n\t%w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	data, err := os.ReadFile(pm.path)
	if err == nil {
		if err := json.Unmarshal(data, pm.data); err != nil {
			return fmt.Errorf("parse %s error:\n\t%w", pm.path, err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s error:\n\t%w", pm.path, err)
	}

	bakData, bakErr := os.ReadFile(pm.path + ".bak")
	if bakErr == nil {
		if err := json.Unmarshal(bakData, pm.data); err != nil {
			return fmt.Errorf("parse %s error:\n\t%w", pm.path+".bak", err)
		}
		return nil
	}
	if !errors.Is(bakErr, os.ErrNotExist) {
		return fmt.Errorf("read %s error:\n\t%w", pm.path+".bak", bakErr)
	}

	// 全新的工作目录：看是否有历史散目录数据可迁移
	if migrated, err := pm.migrateLegacy(); err != nil {
		return err
	} else if migrated {
		return pm.save()
	}
	return nil
}

// save 统一保存：先把当前 data.json 备份为 .bak，再原子写入新文件
func (pm *ProviderManager) save() error {
	data, err := json.MarshalIndent(pm.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provider data error:\n\t%w", err)
	}

	tmp := pm.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write provider data tmp error:\n\t%w", err)
	}
	// 现有文件 -> .bak（保存前备份）
	if _, err := os.Stat(pm.path); err == nil {
		if err := os.Rename(pm.path, pm.path+".bak"); err != nil {
			return fmt.Errorf("backup provider data error:\n\t%w", err)
		}
	}
	if err := os.Rename(tmp, pm.path); err != nil {
		return fmt.Errorf("save provider data error:\n\t%w", err)
	}
	return nil
}

// SaveNow 立即把当前内存态落盘（调用方需在持锁时调用？不，这里自己加锁）
//
// TODO(退出 flush): 服务停止 / 进程退出路径应调用本方法，避免最后窗口内的变更丢失
func (pm *ProviderManager) SaveNow() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.save()
}

// scheduleSave 提交一个延迟保存任务：窗口内多次变更会被节流合并，
// 每次真正落盘的都是当时的完整内存态。
func (pm *ProviderManager) scheduleSave() {
	pm.throttle.Submit(saveKey, saveDelay, func() {
		pm.mu.Lock()
		err := pm.save()
		pm.mu.Unlock()
		if err != nil {
			// TODO(保存失败日志): 延迟落盘失败无人感知，需要留痕（log 或后续通知机制）
		}
	})
}

// migrateLegacy 把历史散目录（providers/<uuid>/*.json 式小文件 + current/last/old）
// 一次性导入 data.json；旧的订阅原文需能解析为 clash 节点，否则跳过该版本。
// 迁移完成后旧目录保留，不删除。返回是否有数据可迁移。调用方需持锁。
func (pm *ProviderManager) migrateLegacy() (bool, error) {
	entries, err := os.ReadDir(pm.ProvidersDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read providers dir error:\n\t%w", err)
	}

	migrated := false
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		uuid := e.Name()
		if len(uuid) < 2 || uuid == "data" { // 防误读隐藏/无关目录
			continue
		}
		dir := filepath.Join(pm.ProvidersDir(), uuid)
		entry := ProviderEntry{
			ProviderConfig: ProviderConfig{Uuid: uuid},
		}
		if content, ok := readFileContent(filepath.Join(dir, "name")); ok {
			entry.Name = content
		}
		if content, ok := readFileContent(filepath.Join(dir, "url")); ok && content != "" {
			entry.Source = SourceURL
			entry.Url = content
		} else {
			entry.Source = SourceUpload
		}
		if content, ok := readFileContent(filepath.Join(dir, "file_name")); ok {
			entry.FileName = content
		}
		if content, ok := readFileContent(filepath.Join(dir, "message")); ok {
			entry.Message = content
		}
		// 订阅版本：current/last/old -> 下标 0/1/2，仅导入能解析为 clash 节点的版本
		for _, v := range []string{versionCurrent, versionLast, versionOld} {
			raw, err := os.ReadFile(filepath.Join(dir, v))
			if err != nil {
				continue
			}
			nodes, err := converter.NodesFromClash(raw)
			if err != nil {
				continue // 旧版本无法解析则跳过（可重新 fetch）
			}
			data, err := json.Marshal(nodes)
			if err != nil {
				continue
			}
			entry.Nodes = append(entry.Nodes, &SubscriptionNode{Data: data})
		}
		pm.data.Providers = append(pm.data.Providers, entry)
		migrated = true
	}
	return migrated, nil
}

func (pm *ProviderManager) ProvidersDir() string {
	return filepath.Join(pm.workingDir, "providers")
}

// ================= provider 元数据 =================

// Add 新建 provider，返回 uuid。source 为 url 时写 url；upload 时不带 url，等待上传。
func (pm *ProviderManager) Add(name string, url string, source string, message string) (string, error) {
	pm.mu.Lock()
	uuid, err := pm.addLocked(name, url, source, message)
	pm.mu.Unlock()
	if err != nil {
		return "", err
	}
	pm.scheduleSave()
	return uuid, nil
}

func (pm *ProviderManager) addLocked(name string, url string, source string, message string) (string, error) {
	if source == "" {
		source = SourceURL
	}
	if err := pm.validateLocked(name, url, source); err != nil {
		return "", err
	}
	if pm.findByNameLocked(name) != "" {
		return "", fmt.Errorf("duplicate provider name '%s'", name)
	}

	entry := ProviderEntry{
		ProviderConfig: ProviderConfig{
			Uuid:    uuid.NewString(),
			Name:    name,
			Url:     url,
			Source:  source,
			Message: message,
		},
	}
	pm.data.Providers = append(pm.data.Providers, entry)
	return entry.Uuid, nil
}

// Update 更新 provider 元数据。source 切为 url 时写 url 并清除 file_name；切为 upload 时清除 url。
func (pm *ProviderManager) Update(uuid string, name string, url string, source string, message string) error {
	pm.mu.Lock()
	err := pm.updateLocked(uuid, name, url, source, message)
	pm.mu.Unlock()
	if err != nil {
		return err
	}
	pm.scheduleSave()
	return nil
}

func (pm *ProviderManager) updateLocked(uuid string, name string, url string, source string, message string) error {
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if source == "" {
		source = SourceURL
	}
	if err := pm.validateLocked(name, url, source); err != nil {
		return err
	}
	if other := pm.findByNameLocked(name); other != "" && other != uuid {
		return fmt.Errorf("duplicate provider name '%s'", name)
	}

	entry := &pm.data.Providers[idx]
	entry.Name = name
	entry.Message = message
	if source == SourceURL {
		entry.Url = url
		entry.FileName = "" // 切回 url 时清除上传文件名
	} else {
		entry.Url = ""
	}
	entry.Source = source
	return nil
}

// SetFileName 记录上传文件名（upload 来源）
func (pm *ProviderManager) SetFileName(uuid string, fileName string) error {
	pm.mu.Lock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		pm.mu.Unlock()
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	pm.data.Providers[idx].FileName = fileName
	pm.mu.Unlock()
	pm.scheduleSave()
	return nil
}

// Delete 删除 provider（含节点数据）。uuid 不存在时不报错。
func (pm *ProviderManager) Delete(uuid string) error {
	pm.mu.Lock()
	idx := pm.indexLocked(uuid)
	if idx >= 0 {
		if pm.data.DefaultProvider == uuid {
			pm.data.DefaultProvider = ""
		}
		pm.data.Providers = append(pm.data.Providers[:idx], pm.data.Providers[idx+1:]...)
	}
	pm.mu.Unlock()
	pm.scheduleSave()
	return nil
}

// Get 读取 provider 元数据；不存在返回 nil
func (pm *ProviderManager) Get(uuid string) *ProviderConfig {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		return nil
	}
	conf := pm.data.Providers[idx].ProviderConfig
	conf.Default = uuid == pm.data.DefaultProvider
	return &conf
}

// List 列出所有 provider（按名称排序）
func (pm *ProviderManager) List() []ProviderConfig {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	list := make([]ProviderConfig, 0, len(pm.data.Providers))
	for _, e := range pm.data.Providers {
		conf := e.ProviderConfig
		conf.Default = conf.Uuid == pm.data.DefaultProvider
		list = append(list, conf)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

// Exists 判断 provider 是否存在
func (pm *ProviderManager) Exists(uuid string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.indexLocked(uuid) >= 0
}

// IsUploadSource 判断 provider 是否为上传来源
func (pm *ProviderManager) IsUploadSource(uuid string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		return false
	}
	return pm.data.Providers[idx].Source == SourceUpload
}

// 以下 *Locked 方法假定调用方已持有 pm.mu
func (pm *ProviderManager) indexLocked(uuid string) int {
	for i := range pm.data.Providers {
		if pm.data.Providers[i].Uuid == uuid {
			return i
		}
	}
	return -1
}

func (pm *ProviderManager) findByNameLocked(name string) string {
	for i := range pm.data.Providers {
		if pm.data.Providers[i].Name == name {
			return pm.data.Providers[i].Uuid
		}
	}
	return ""
}

func (pm *ProviderManager) validateLocked(name string, url string, source string) error {
	if source != SourceURL && source != SourceUpload {
		return fmt.Errorf("invalid source '%s', must be '%s' or '%s'", source, SourceURL, SourceUpload)
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if source == SourceURL {
		if url == "" {
			return fmt.Errorf("url is required when source is '%s'", SourceURL)
		}
		if !isHTTPURL(url) {
			return fmt.Errorf("invalid url '%s', must be http(s) url", url)
		}
	}
	return nil
}

// ================= 默认 provider =================

// SetDefaultProvider 将指定 provider 设为默认
func (pm *ProviderManager) SetDefaultProvider(uuid string) error {
	pm.mu.Lock()
	if pm.indexLocked(uuid) < 0 {
		pm.mu.Unlock()
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	pm.data.DefaultProvider = uuid
	pm.mu.Unlock()
	pm.scheduleSave()
	return nil
}

// DefaultProviderUUID 返回默认 provider uuid；未设置时取第一个兜底，无 provider 返回空串
func (pm *ProviderManager) DefaultProviderUUID() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.indexLocked(pm.data.DefaultProvider) >= 0 {
		return pm.data.DefaultProvider
	}
	if len(pm.data.Providers) > 0 {
		return pm.data.Providers[0].Uuid
	}
	return ""
}

// ================= 节点版本（订阅内容） =================

// SaveNodes 保存新一版节点 outbounds JSON（数组），滚动保留最多 3 份
func (pm *ProviderManager) SaveNodes(uuid string, nodesJSON []byte) error {
	pm.mu.Lock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		pm.mu.Unlock()
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	if !json.Valid(nodesJSON) {
		pm.mu.Unlock()
		return fmt.Errorf("invalid json data")
	}
	entry := &pm.data.Providers[idx]
	data := append([]byte(nil), nodesJSON...)
	nodes := []*SubscriptionNode{{Data: data}}
	for _, n := range entry.Nodes {
		if len(nodes) >= 3 {
			break
		}
		nodes = append(nodes, n)
	}
	entry.Nodes = nodes
	pm.mu.Unlock()
	pm.scheduleSave()
	return nil
}

// ReadSubscription 读取当前节点内容（outbounds JSON 数组）
func (pm *ProviderManager) ReadSubscription(uuid string) ([]byte, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		return nil, fmt.Errorf("no provider with uuid: %s", uuid)
	}
	nodes := pm.data.Providers[idx].Nodes
	if len(nodes) == 0 {
		return nil, fmt.Errorf("provider '%s' has no subscription data, fetch or upload first", uuid)
	}
	return append([]byte(nil), nodes[0].Data...), nil
}

// SubscriptionVersions 返回节点数据存在的历史版本（current/last/old 顺序）
func (pm *ProviderManager) SubscriptionVersions(uuid string) ([]string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		return nil, fmt.Errorf("no provider with uuid: %s", uuid)
	}
	count := len(pm.data.Providers[idx].Nodes)
	versions := make([]string, 0, count)
	for _, v := range []string{subVersionCurrent, subVersionLast, subVersionOld} {
		if subVersionIndex[v] < count {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

// RestoreSubscription 用指定历史版本覆盖当前节点（会再次滚动保留一层历史）
func (pm *ProviderManager) RestoreSubscription(uuid, version string) error {
	pm.mu.Lock()
	idx := pm.indexLocked(uuid)
	if idx < 0 {
		pm.mu.Unlock()
		return fmt.Errorf("no provider with uuid: %s", uuid)
	}
	pos, ok := subVersionIndex[version]
	if !ok {
		pm.mu.Unlock()
		return fmt.Errorf("invalid version: %s", version)
	}
	nodes := pm.data.Providers[idx].Nodes
	if pos >= len(nodes) {
		pm.mu.Unlock()
		return fmt.Errorf("version '%s' not exists", version)
	}
	data := append([]byte(nil), nodes[pos].Data...)
	pm.mu.Unlock()
	return pm.SaveNodes(uuid, data)
}
