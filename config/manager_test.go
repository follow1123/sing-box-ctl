package config_test

import (
	"os"
	"testing"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/stretchr/testify/require"
)

func TestCreateConfigManagerSucceed(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
}

func TestLoadConfigSucceed(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	conf, err := cm.Load()
	r.NoError(err)
	r.NotNil(conf)
}

func TestLoadConfigFailed(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	t.Run("config file not exists", func(t *testing.T) {
		err := os.Remove(cm.GetConfigFile())
		r.NoError(err)
		_, err = cm.Load()
		r.ErrorContains(err, "load config error")
	})
	t.Run("config file is not yaml", func(t *testing.T) {
		err := os.WriteFile(cm.GetConfigFile(), []byte("fdsakfjsdkf"), 0660)
		r.NoError(err)
		_, err = cm.Load()
		r.ErrorContains(err, "parse config error")
	})
}

func TestChangeConfig(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	t.Run("change default config", func(t *testing.T) {
		pathMap := map[string]any{
			"$.singbox.check_cmd":        "echo 1234",
			"$.singbox.options.mode":     "tun",
			"$.services.updater.enabled": false,
		}
		err := cm.ChangeConfig(pathMap)
		r.NoError(err)
		conf, err := cm.Load()
		r.NoError(err)
		r.Equal(pathMap["$.singbox.check_cmd"], conf.SingBox.CheckCommand)
		r.Equal(pathMap["$.singbox.options.mode"], conf.SingBox.Options.Mode)
		r.Equal(pathMap["$.services.updater.enabled"], conf.Services.Updater.Enabled)
	})

	t.Run("change empty config", func(t *testing.T) {
		err := os.WriteFile(cm.GetConfigFile(), []byte(""), 0660)
		r.NoError(err)
		data, err := os.ReadFile(cm.GetConfigFile())
		r.NoError(err)
		r.Equal("", string(data), "truncate config error")
		pathMap := map[string]any{
			"$.singbox.check_cmd":        "echo 1234",
			"$.singbox.options.mode":     "tun",
			"$.services.updater.enabled": false,
		}
		err = cm.ChangeConfig(pathMap)
		r.NoError(err)

		conf, err := cm.Load()
		r.NoError(err)
		r.Equal(pathMap["$.singbox.check_cmd"], conf.SingBox.CheckCommand)
		r.Equal(pathMap["$.singbox.options.mode"], conf.SingBox.Options.Mode)
		r.Equal(pathMap["$.services.updater.enabled"], conf.Services.Updater.Enabled)
	})
}

func TestRestoreConfig(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	pathMap := map[string]any{
		"$.singbox.check_cmd": "echo 1234",
	}
	cm.ChangeConfig(pathMap)
	config, err := cm.Load()
	r.NoError(err)
	r.Equal(pathMap["$.singbox.check_cmd"], config.SingBox.CheckCommand)

	err = cm.RestoreDefaultConfig()
	r.NoError(err)
	config, err = cm.Load()
	r.NoError(err)
	r.NotEqual(pathMap["$.singbox.check_cmd"], config.SingBox.CheckCommand)
}

func TestRestoreTemplate(t *testing.T) {
	r := require.New(t)
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	tmplData := `
	{
		"name": "dsafds",
		"age":1243
	}
	`
	err := os.WriteFile(cm.GetTemplateFile(), []byte(tmplData), 0660)
	r.NoError(err)
	err = cm.RestoreDefaultTemplate()
	r.NoError(err)
	data, err := os.ReadFile(cm.GetTemplateFile())
	r.NoError(err)
	r.NotEqual(tmplData, string(data))
}
