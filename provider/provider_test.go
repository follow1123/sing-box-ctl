package provider_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/stretchr/testify/require"
)

// newProvider 构造模板文件系统 Provider（供模板相关测试使用）
func newProvider(t *testing.T) (*provider.Provider, string) {
	t.Helper()
	workingDir := t.TempDir()
	p, err := provider.New(workingDir)
	require.NoError(t, err)
	return p, workingDir
}

// newManager 构造 provider 数据管理器（working_dir/providers/data.json）
func newManager(t *testing.T) (*provider.ProviderManager, string) {
	t.Helper()
	workingDir := t.TempDir()
	pm, err := provider.NewManager(workingDir)
	require.NoError(t, err)
	return pm, workingDir
}

// nodesJSON 生成一份最小的节点 outbounds JSON（数组）
func nodesJSON(t *testing.T, tags ...string) []byte {
	t.Helper()
	outbounds := make([]map[string]any, 0, len(tags))
	for i, tag := range tags {
		outbounds = append(outbounds, map[string]any{"type": "shadowsocks", "tag": tag, "server": "1.2.3.4", "port": 8000 + i})
	}
	data, err := json.Marshal(outbounds)
	require.NoError(t, err)
	return data
}

func TestAdd(t *testing.T) {
	t.Run("add url source", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
		require.NoError(t, err)
		list := pm.List()
		require.Len(t, list, 1)
		require.NotEmpty(t, list[0].Uuid)
		require.Equal(t, "aaa", list[0].Name)
		require.Equal(t, "http://localhost:8752", list[0].Url)
		require.Equal(t, provider.SourceURL, list[0].Source)
	})

	t.Run("add upload source", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("bbb", "", provider.SourceUpload, "upload message")
		require.NoError(t, err)
		list := pm.List()
		require.Len(t, list, 1)
		require.Equal(t, provider.SourceUpload, list[0].Source)
		require.Empty(t, list[0].Url)
		require.Equal(t, "upload message", list[0].Message)
	})

	t.Run("invalid source", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("aaa", "http://localhost:8752", "invalid", "")
		require.ErrorContains(t, err, "invalid source")
	})

	t.Run("url required for url source", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("aaa", "", provider.SourceURL, "")
		require.ErrorContains(t, err, "url is required")
	})

	t.Run("name required", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("", "http://localhost:8752", provider.SourceURL, "")
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("duplicate provider name", func(t *testing.T) {
		pm, _ := newManager(t)
		_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
		require.NoError(t, err)
		_, err = pm.Add("aaa", "http://localhost:8753", provider.SourceURL, "")
		require.ErrorContains(t, err, "duplicate provider name")
	})
}

func TestUpdate(t *testing.T) {
	pm, _ := newManager(t)
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid

	err = pm.Update(uuid, "aaa2", "http://localhost:8753", provider.SourceUpload, "new msg")
	require.NoError(t, err)
	prov := pm.Get(uuid)
	require.Equal(t, "aaa2", prov.Name)
	require.Equal(t, provider.SourceUpload, prov.Source)
	require.Empty(t, prov.Url)
	require.Equal(t, "new msg", prov.Message)

	// 上传文件名记录后，切回 url 应清除
	require.NoError(t, pm.SetFileName(uuid, "test.yaml"))
	err = pm.Update(uuid, "aaa2", "http://localhost:8753", provider.SourceURL, "new msg")
	require.NoError(t, err)
	prov = pm.Get(uuid)
	require.Equal(t, provider.SourceURL, prov.Source)
	require.Empty(t, prov.FileName)

	t.Run("update unknown uuid", func(t *testing.T) {
		err := pm.Update("unknown", "x", "http://x", provider.SourceURL, "")
		require.ErrorContains(t, err, "no provider with uuid")
	})
}

func TestDelete(t *testing.T) {
	pm, _ := newManager(t)
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	_, err = pm.Add("bbb", "http://localhost:8753", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid

	require.NoError(t, pm.Delete(uuid))
	require.Len(t, pm.List(), 1)
	require.Nil(t, pm.Get(uuid))

	// 删除不存在的 uuid 不报错
	require.NoError(t, pm.Delete("unknown"))
}

func TestGet(t *testing.T) {
	pm, _ := newManager(t)
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid
	require.Equal(t, "aaa", pm.Get(uuid).Name)
	require.Nil(t, pm.Get("unknown"))
}

func TestPersistence(t *testing.T) {
	// 数据保存到 providers/data.json，重新加载后仍能读到
	pm, workingDir := newManager(t)
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "hello")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid
	require.NoError(t, pm.SetFileName(uuid, "sub.yaml"))
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-a", "节点-b")))
	require.NoError(t, pm.SaveNow()) // 变更默认节流延迟落盘，验证前立即写一次
	require.NoError(t, pm.SaveNow()) // 再次保存应把旧 data.json 备份为 .bak

	pm2, err := provider.NewManager(workingDir)
	require.NoError(t, err)
	list := pm2.List()
	require.Len(t, list, 1)
	require.Equal(t, "aaa", list[0].Name)
	require.Equal(t, "http://localhost:8752", list[0].Url)
	require.Equal(t, "hello", list[0].Message)
	require.Equal(t, "sub.yaml", list[0].FileName)

	data, err := pm2.ReadSubscription(uuid)
	require.NoError(t, err)
	var nodes []map[string]any
	require.NoError(t, json.Unmarshal(data, &nodes))
	require.Len(t, nodes, 2)

	// data.json 与 .bak 均在；内容可解析（多次变更后 .bak 为上次成功的旧数据）
	require.FileExists(t, filepath.Join(workingDir, "providers", "data.json"))
	require.FileExists(t, filepath.Join(workingDir, "providers", "data.json.bak"))
}

func TestSaveNodesAndVersions(t *testing.T) {
	pm, _ := newManager(t)
	_, err := pm.Add("aaa", "", provider.SourceUpload, "")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid

	// 首次保存只有 current
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-a")))
	vers, err := pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current"}, vers)

	// 第二次 current/last
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-b")))
	vers, err = pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last"}, vers)

	// 第三次三层齐
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-c")))
	vers, err = pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last", "old"}, vers)

	// 第四次仍最多三层，最早被挤出
	require.NoError(t, pm.SaveNodes(uuid, nodesJSON(t, "节点-d")))
	data, err := pm.ReadSubscription(uuid)
	require.NoError(t, err)
	var nodes []map[string]any
	require.NoError(t, json.Unmarshal(data, &nodes))
	require.Equal(t, "节点-d", nodes[0]["tag"])

	// 还原 old（节点-b）：current 变为它，滚动保留一层
	require.NoError(t, pm.RestoreSubscription(uuid, "old"))
	data, err = pm.ReadSubscription(uuid)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &nodes))
	require.Equal(t, "节点-b", nodes[0]["tag"])
	vers, err = pm.SubscriptionVersions(uuid)
	require.NoError(t, err)
	require.Len(t, vers, 3, "还原后仍保留三层历史")

	require.Error(t, pm.RestoreSubscription(uuid, "not-a-version"))
}

func TestReadSubscriptionEmpty(t *testing.T) {
	pm, _ := newManager(t)
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := pm.List()[0].Uuid
	_, err = pm.ReadSubscription(uuid)
	require.ErrorContains(t, err, "no subscription data")

	_, err = pm.ReadSubscription("unknown")
	require.ErrorContains(t, err, "no provider with uuid")
}

func TestDefaultProvider(t *testing.T) {
	pm, _ := newManager(t)
	// 空时无默认
	require.Empty(t, pm.DefaultProviderUUID())
	_, err := pm.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	// 未设置默认时取第一个兜底
	uuid := pm.List()[0].Uuid
	require.Equal(t, uuid, pm.DefaultProviderUUID())
	// 显式设置
	require.NoError(t, pm.SetDefaultProvider(uuid))
	require.Equal(t, uuid, pm.DefaultProviderUUID())
	// 删除默认后清空回退
	require.NoError(t, pm.Delete(uuid))
	require.Empty(t, pm.DefaultProviderUUID())
}

// TestMigrateLegacy 历史散目录（providers/<uuid>/…）应一次性导入 data.json
func TestMigrateLegacy(t *testing.T) {
	workingDir := t.TempDir()
	// 手工构造旧目录结构
	dir := filepath.Join(workingDir, "providers", "legacy-1")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "name"), []byte("old-name"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "url"), []byte("http://example.com/sub"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "message"), []byte("migrate me"), 0600))
	clash := `proxies:
  - name: 节点-1
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-128-gcm
    password: secret
  - name: 节点-2
    type: trojan
    server: 2.2.2.2
    port: 443
    password: x
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "current"), []byte(clash), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "last"), []byte(clash), 0600))

	pm, err := provider.NewManager(workingDir)
	require.NoError(t, err)
	require.NoError(t, err)

	list := pm.List()
	require.Len(t, list, 1)
	prov := list[0]
	require.Equal(t, "old-name", prov.Name)
	require.Equal(t, "http://example.com/sub", prov.Url)
	require.Equal(t, provider.SourceURL, prov.Source)
	require.Equal(t, "migrate me", prov.Message)

	// 两个版本成功导入（current/last）
	vers, err := pm.SubscriptionVersions(prov.Uuid)
	require.NoError(t, err)
	require.Equal(t, []string{"current", "last"}, vers)
	data, err := pm.ReadSubscription(prov.Uuid)
	require.NoError(t, err)
	var nodes []map[string]any
	require.NoError(t, json.Unmarshal(data, &nodes))
	require.Len(t, nodes, 2)

	// data.json 已生成
	require.FileExists(t, filepath.Join(workingDir, "providers", "data.json"))
}
