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
	workingDir := t.TempDir()
	p, err := provider.New(workingDir)
	require.NoError(t, err)
	return p, workingDir
}

func TestAdd(t *testing.T) {
	t.Run("add url source", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
		require.NoError(t, err)
		list := p.List()
		require.Len(t, list, 1)
		require.NotEmpty(t, list[0].Uuid)
		require.Equal(t, "aaa", list[0].Name)
		require.Equal(t, "http://localhost:8752", list[0].Url)
		require.Equal(t, provider.SourceURL, list[0].Source)
	})

	t.Run("add upload source", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("bbb", "", provider.SourceUpload, "upload message")
		require.NoError(t, err)
		list := p.List()
		require.Len(t, list, 1)
		require.Equal(t, provider.SourceUpload, list[0].Source)
		require.Empty(t, list[0].Url)
		require.Equal(t, "upload message", list[0].Message)
	})

	t.Run("source inferred from files", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("url-x", "http://localhost:8752", provider.SourceURL, "")
		require.NoError(t, err)
		_, err = p.Add("up-x", "", provider.SourceUpload, "")
		require.NoError(t, err)
		for _, info := range p.List() {
			if info.Name == "url-x" {
				require.Equal(t, provider.SourceURL, info.Source)
			} else {
				require.Equal(t, provider.SourceUpload, info.Source)
			}
		}
	})

	t.Run("invalid source", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("aaa", "http://localhost:8752", "invalid", "")
		require.ErrorContains(t, err, "invalid source")
	})

	t.Run("url required for url source", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("aaa", "", provider.SourceURL, "")
		require.ErrorContains(t, err, "url is required")
	})

	t.Run("name required", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("", "http://localhost:8752", provider.SourceURL, "")
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("duplicate provider name", func(t *testing.T) {
		p, _ := newProvider(t)
		_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
		require.NoError(t, err)
		_, err = p.Add("aaa", "http://localhost:8753", provider.SourceURL, "")
		require.ErrorContains(t, err, "duplicate provider name")
	})
}

func TestUpdate(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := p.List()[0].Uuid

	err = p.Update(uuid, "aaa2", "http://localhost:8753", provider.SourceUpload, "new msg")
	require.NoError(t, err)
	prov := p.Get(uuid)
	require.Equal(t, "aaa2", prov.Name)
	require.Equal(t, provider.SourceUpload, prov.Source)
	require.Empty(t, prov.Url)
	require.Equal(t, "new msg", prov.Message)

	// 上传文件名记录后，切回 url 应清除
	require.NoError(t, p.SetFileName(uuid, "test.yaml"))
	err = p.Update(uuid, "aaa2", "http://localhost:8753", provider.SourceURL, "new msg")
	require.NoError(t, err)
	prov = p.Get(uuid)
	require.Equal(t, provider.SourceURL, prov.Source)
	require.Empty(t, prov.FileName)

	t.Run("update unknown uuid", func(t *testing.T) {
		err := p.Update("unknown", "x", "http://x", provider.SourceURL, "")
		require.ErrorContains(t, err, "no provider with uuid")
	})
}

func TestDelete(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	_, err = p.Add("bbb", "http://localhost:8753", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := p.List()[0].Uuid

	require.NoError(t, p.Delete(uuid))
	require.Len(t, p.List(), 1)
	require.Nil(t, p.Get(uuid))

	// 删除不存在的 uuid 不报错
	require.NoError(t, p.Delete("unknown"))
}

func TestGet(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := p.List()[0].Uuid
	require.Equal(t, "aaa", p.Get(uuid).Name)
	require.Nil(t, p.Get("unknown"))
}

func TestPersistence(t *testing.T) {
	// 元数据在文件系统，重新加载后仍能读到
	p, workingDir := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "hello")
	require.NoError(t, err)
	uuid := p.List()[0].Uuid
	require.NoError(t, p.SetFileName(uuid, "sub.yaml"))

	p2, err := provider.New(workingDir)
	require.NoError(t, err)
	list := p2.List()
	require.Len(t, list, 1)
	require.Equal(t, "aaa", list[0].Name)
	require.Equal(t, "http://localhost:8752", list[0].Url)
	require.Equal(t, "hello", list[0].Message)
	require.Equal(t, "sub.yaml", list[0].FileName)
}

func TestSaveSubscription(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
	uuid := p.List()[0].Uuid
	dir := p.SubscriptionDir(uuid)

	require.NoError(t, p.SaveSubscription(uuid, []byte("data1")))
	require.FileExists(t, filepath.Join(dir, "current"))
	require.NoFileExists(t, filepath.Join(dir, "last"))

	require.NoError(t, p.SaveSubscription(uuid, []byte("data2")))
	assertFileContent(t, filepath.Join(dir, "current"), "data2")
	assertFileContent(t, filepath.Join(dir, "last"), "data1")

	require.NoError(t, p.SaveSubscription(uuid, []byte("data3")))
	assertFileContent(t, filepath.Join(dir, "current"), "data3")
	assertFileContent(t, filepath.Join(dir, "last"), "data2")
	assertFileContent(t, filepath.Join(dir, "old"), "data1")
}

func TestReadSubscription(t *testing.T) {
	p, _ := newProvider(t)
	_, err := p.Add("aaa", "http://localhost:8752", provider.SourceURL, "")
	require.NoError(t, err)
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
