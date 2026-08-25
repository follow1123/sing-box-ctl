package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/stretchr/testify/require"
)

func newProvider(t *testing.T) (*provider.Provider, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	workingDir := filepath.Join(t.TempDir(), "working_dir")
	content := `{"working_dir": "` + workingDir + `"}`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0660))
	p, err := provider.New(configPath)
	require.NoError(t, err)
	return p, configPath
}

func TestInitDefaultTemplate(t *testing.T) {
	t.Run("empty creates default template", func(t *testing.T) {
		p, configPath := newProvider(t)
		require.NoError(t, os.MkdirAll(p.TemplatesDir(), 0700))
		conf, err := config.New(configPath)
		require.NoError(t, err)
		infos, err := p.ListTemplates()
		require.NoError(t, err)
		require.Len(t, infos, 1)
		require.True(t, infos[0].Default)
		require.Equal(t, conf.TemplatesDir, p.TemplatesDir())
	})

	t.Run("templates without default get first marked", func(t *testing.T) {
		p, configPath := newProvider(t)
		// 手工创建两个模板，无 default 标记（模拟 default 文件被手动删除）
		uuidA, err := p.AddTemplate("模板A")
		require.NoError(t, err)
		require.NoError(t, p.SaveTemplate(uuidA, []byte("{}")))
		uuidB, err := p.AddTemplate("模板B")
		require.NoError(t, err)
		require.NoError(t, p.SaveTemplate(uuidB, []byte("{}")))

		_, err = config.New(configPath)
		require.NoError(t, err)

		infos, err := p.ListTemplates()
		require.NoError(t, err)
		require.Len(t, infos, 2)
		// 第一个（名称最小）被补为默认，且唯一
		require.True(t, infos[0].Default)
		require.False(t, infos[1].Default)
		require.Equal(t, "模板A", infos[0].Name)
	})

	t.Run("existing default preserved", func(t *testing.T) {
		p, configPath := newProvider(t)
		uuidA, err := p.AddTemplate("模板A")
		require.NoError(t, err)
		require.NoError(t, p.SaveTemplate(uuidA, []byte("{}")))
		uuidB, err := p.AddTemplate("模板B")
		require.NoError(t, err)
		require.NoError(t, p.SaveTemplate(uuidB, []byte("{}")))
		require.NoError(t, p.SetDefaultTemplate(uuidB))

		_, err = config.New(configPath)
		require.NoError(t, err)

		infos, err := p.ListTemplates()
		require.NoError(t, err)
		require.Len(t, infos, 2)
		require.Equal(t, uuidB, infos[0].Uuid)
		require.True(t, infos[0].Default)
	})
}
