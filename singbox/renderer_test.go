package singbox_test

import (
	"os"
	"strings"
	"testing"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
	"github.com/stretchr/testify/require"
)

func confManager(r *require.Assertions, t *testing.T) *config.ConfigManager {
	confHome := t.TempDir()
	cm := config.NewConfManager(confHome)
	r.NotNil(cm)
	return cm
}

func TestCreateConfigRendererSucceed(t *testing.T) {
	r := require.New(t)
	cm := confManager(r, t)
	tl := logger.NewTestLogger(t)
	cr, err := singbox.NewConfRenderer(tl, cm.GetTemplateFile())
	r.NoError(err)
	r.NotNil(cr)
}

func TestCreateConfigRendererFailure(t *testing.T) {
	t.Run("template file not exists", func(t *testing.T) {
		r := require.New(t)
		cm := confManager(r, t)
		err := os.Remove(cm.GetTemplateFile())
		r.NoError(err)

		tl := logger.NewTestLogger(t)
		_, err = singbox.NewConfRenderer(tl, cm.GetTemplateFile())
		r.ErrorContains(err, "read template file error")
	})
	t.Run("invalid template file", func(t *testing.T) {
		r := require.New(t)
		cm := confManager(r, t)
		err := os.WriteFile(cm.GetTemplateFile(), []byte(`{"name": "{{.aaa}"}`), 0660)
		r.NoError(err)

		tl := logger.NewTestLogger(t)
		_, err = singbox.NewConfRenderer(tl, cm.GetTemplateFile())
		r.ErrorContains(err, "parse template error")
	})
}

func TestRenderSucceed(t *testing.T) {
	r := require.New(t)
	cm := confManager(r, t)
	tl := logger.NewTestLogger(t)
	cr, err := singbox.NewConfRenderer(tl, cm.GetTemplateFile())
	r.NoError(err)
	_, err = cr.Render(singbox.NewPartialConfig(), &singbox.Options{})
	r.NoError(err)
}

func TestRenderFailure(t *testing.T) {
	t.Run("undefined key", func(t *testing.T) {
		r := require.New(t)
		cm := confManager(r, t)
		tmplData, err := os.ReadFile(cm.GetTemplateFile())
		r.NoError(err)

		newData := strings.ReplaceAll(string(tmplData), ".Pc", "")

		err = os.WriteFile(cm.GetTemplateFile(), []byte(newData), 0660)
		r.NoError(err)

		tl := logger.NewTestLogger(t)
		cr, err := singbox.NewConfRenderer(tl, cm.GetTemplateFile())
		r.NoError(err)
		_, err = cr.Render(singbox.NewPartialConfig(), &singbox.Options{})
		r.ErrorContains(err, "render template error")
	})
}
