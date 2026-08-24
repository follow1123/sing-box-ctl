package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/stretchr/testify/require"
)

func newProvider(t *testing.T) (*provider.Provider, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	// 写一个含 working_dir 的配置，避免相对路径推导问题
	workingDir := t.TempDir()
	content := `{"working_dir": "` + workingDir + `", "providers": []}`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0660))
	p, err := provider.New(configPath)
	require.NoError(t, err)
	return p, configPath
}

func TestAdd(t *testing.T) {
	t.Run("add success and generate uuid", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
		list := p.List()
		require.Len(t, list, 1)
		require.NotEmpty(t, list[0].Uuid)
		require.Equal(t, "aaa", list[0].Name)
		require.Equal(t, provider.SourceURL, list[0].Source)
	})

	t.Run("add with upload source no url", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("bbb", "", provider.SourceUpload, "upload message"))
		list := p.List()
		require.Len(t, list, 1)
		require.Equal(t, provider.SourceUpload, list[0].Source)
		require.Equal(t, "upload message", list[0].Message)
	})

	t.Run("invalid source", func(t *testing.T) {
		p, _ := newProvider(t)
		err := p.Add("aaa", "http://localhost:8752", "invalid", "")
		require.ErrorContains(t, err, "invalid source")
	})

	t.Run("url required for url source", func(t *testing.T) {
		p, _ := newProvider(t)
		err := p.Add("aaa", "", provider.SourceURL, "")
		require.ErrorContains(t, err, "url is required")
	})

	t.Run("name required", func(t *testing.T) {
		p, _ := newProvider(t)
		err := p.Add("", "http://localhost:8752", provider.SourceURL, "")
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("duplicate provider name", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
		err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
		require.ErrorContains(t, err, "duplicate provider name")
	})
}

func TestUpdate(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	uuid := p.List()[0].Uuid

	require.NoError(t, p.Update(uuid, "aaa2", "http://localhost:8753", provider.SourceUpload, "new msg"))
	prov := p.Get(uuid)
	require.Equal(t, "aaa2", prov.Name)
	require.Equal(t, "http://localhost:8753", prov.Url)
	require.Equal(t, provider.SourceUpload, prov.Source)
	require.Equal(t, "new msg", prov.Message)

	t.Run("update unknown uuid", func(t *testing.T) {
		err := p.Update("unknown", "x", "http://x", provider.SourceURL, "")
		require.ErrorContains(t, err, "no provider with uuid")
	})
}

func TestDelete(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	require.NoError(t, p.Add("bbb", "http://localhost:8753", provider.SourceURL, ""))
	uuid := p.List()[0].Uuid

	require.NoError(t, p.Delete(uuid))
	require.Len(t, p.List(), 1)
	require.Nil(t, p.Get(uuid))

	// 删除不存在的 uuid 不报错
	require.NoError(t, p.Delete("unknown"))
}

func TestGet(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	uuid := p.List()[0].Uuid
	require.Equal(t, "aaa", p.Get(uuid).Name)
	require.Nil(t, p.Get("unknown"))
}

func TestSave(t *testing.T) {
	p, configPath := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	require.NoError(t, p.Save())

	// 重新加载，验证持久化
	p2, err := provider.New(configPath)
	require.NoError(t, err)
	require.Len(t, p2.List(), 1)
	require.Equal(t, "aaa", p2.List()[0].Name)
	require.NotEmpty(t, p2.List()[0].Uuid)
}

func TestSaveSubscription(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	uuid := p.List()[0].Uuid
	dir := p.SubscriptionDir(uuid)

	// 第一次保存
	require.NoError(t, p.SaveSubscription(uuid, []byte("data1")))
	require.FileExists(t, filepath.Join(dir, "current"))
	require.NoFileExists(t, filepath.Join(dir, "last"))

	// 第二次：current→last
	require.NoError(t, p.SaveSubscription(uuid, []byte("data2")))
	assertFileContent(t, filepath.Join(dir, "current"), "data2")
	assertFileContent(t, filepath.Join(dir, "last"), "data1")
	require.NoFileExists(t, filepath.Join(dir, "old"))

	// 第三次：current→last→old 滚动
	require.NoError(t, p.SaveSubscription(uuid, []byte("data3")))
	assertFileContent(t, filepath.Join(dir, "current"), "data3")
	assertFileContent(t, filepath.Join(dir, "last"), "data2")
	assertFileContent(t, filepath.Join(dir, "old"), "data1")

	// 第四次：old 被覆盖丢弃
	require.NoError(t, p.SaveSubscription(uuid, []byte("data4")))
	assertFileContent(t, filepath.Join(dir, "current"), "data4")
	assertFileContent(t, filepath.Join(dir, "last"), "data3")
	assertFileContent(t, filepath.Join(dir, "old"), "data2")
}

func TestReadSubscription(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752", provider.SourceURL, ""))
	uuid := p.List()[0].Uuid
	require.NoError(t, p.SaveSubscription(uuid, []byte("hello")))
	data, err := p.ReadSubscription(uuid)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))

	_, err = p.ReadSubscription("unknown")
	require.Error(t, err)
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, expected, string(data))
}
