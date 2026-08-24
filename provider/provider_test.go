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
	p, err := provider.New(configPath)
	require.NoError(t, err)
	return p, configPath
}

func TestInit(t *testing.T) {
	_, err := provider.New(filepath.Join(t.TempDir(), "config.json"))
	require.NoError(t, err)
}

func TestAdd(t *testing.T) {
	t.Run("add first is default", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		pd := p.Get("aaa")
		defaultProvider := p.GetDefault()
		require.Equal(t, pd.Name, defaultProvider.Name)
	})
	t.Run("added later is not the default", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		require.NoError(t, p.Add("bbb", "http://localhost:8754"))
		pd := p.Get("bbb")
		defaultProvider := p.GetDefault()
		require.NotEqual(t, pd.Name, defaultProvider.Name)
	})

	t.Run("duplicate provider name", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		require.ErrorContains(t, p.Add("aaa", "http://localhost:8752"), "duplicate provider name")
	})
}

func TestSetDefault(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752"))
	defProvider := p.GetDefault()
	require.Equal(t, "aaa", defProvider.Name)
	require.NoError(t, p.Add("bbb", "http://localhost:8754"))
	p.SetDefault("bbb")
	defProvider = p.GetDefault()
	require.Equal(t, "bbb", defProvider.Name)
}

func TestDelete(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		require.NoError(t, p.Add("bbb", "http://localhost:8753"))
		require.NoError(t, p.Add("ccc", "http://localhost:8754"))
		require.NoError(t, p.Add("ddd", "http://localhost:8755"))
		require.NoError(t, p.Delete("ccc"))
		pds := p.List()
		require.Equal(t, "aaa", pds[0].Name)
		require.Equal(t, "bbb", pds[1].Name)
		require.Equal(t, "ddd", pds[2].Name)
	})
	t.Run("delete first if is default", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		require.NoError(t, p.Add("bbb", "http://localhost:8753"))
		require.NoError(t, p.Add("ccc", "http://localhost:8754"))
		require.NoError(t, p.Add("ddd", "http://localhost:8755"))
		require.NoError(t, p.Delete("aaa"))
		defProvider := p.GetDefault()
		require.Equal(t, "bbb", defProvider.Name)
	})
	t.Run("delete last if is default", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		require.NoError(t, p.Add("bbb", "http://localhost:8753"))
		require.NoError(t, p.Add("ccc", "http://localhost:8754"))
		require.NoError(t, p.Add("ddd", "http://localhost:8755"))
		p.SetDefault("ddd")
		require.NoError(t, p.Delete("ddd"))
		defProvider := p.GetDefault()
		require.Equal(t, "ccc", defProvider.Name)
	})
	t.Run("delete last one if is default", func(t *testing.T) {
		p, _ := newProvider(t)
		require.NoError(t, p.Add("aaa", "http://localhost:8752"))
		name := "aaa"
		require.NoError(t, p.Delete(name))
		require.Nil(t, p.Get(name))
	})
}

func TestUpdate(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		p, configPath := newProvider(t)
		err := os.WriteFile(configPath, []byte(`{"providers":[{"name": "aaa","url":"http://localhost:8903"}]}`), 0660)
		require.NoError(t, err)
		p, err = provider.New(configPath)
		require.NoError(t, err)
		name := "aaa"
		newUrl := "http://localhost:8752"
		err = p.Update(name, newUrl)
		require.NoError(t, err)
		data := p.Get(name)
		require.Equal(t, newUrl, data.Url)
	})
	t.Run("no providers", func(t *testing.T) {
		p, _ := newProvider(t)
		err := p.Update("aaa", "http://localhost:8752")
		require.ErrorContains(t, err, "no provider")
	})
}

func TestGet(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		p, configPath := newProvider(t)
		err := os.WriteFile(configPath, []byte(`{"providers":[{"name": "aaa","url":"http://localhost:8903"}]}`), 0660)
		require.NoError(t, err)
		p, err = provider.New(configPath)
		require.NoError(t, err)
		data := p.Get("aaa")
		require.Equal(t, "aaa", data.Name)
		require.Equal(t, "http://localhost:8903", data.Url)
	})
	t.Run("no providers", func(t *testing.T) {
		p, _ := newProvider(t)
		require.Nil(t, p.Get("aaa"))
	})
}

func TestList(t *testing.T) {
	p, _ := newProvider(t)
	require.NoError(t, p.Add("aaa", "http://localhost:8752"))
	require.NoError(t, p.Add("bbb", "http://localhost:8753"))
	require.NoError(t, p.Add("ccc", "http://localhost:8754"))
	require.NoError(t, p.Add("ddd", "http://localhost:8755"))
	pds := p.List()
	require.Equal(t, "aaa", pds[0].Name)
	require.Equal(t, "bbb", pds[1].Name)
	require.Equal(t, "ccc", pds[2].Name)
	require.Equal(t, "ddd", pds[3].Name)
}
