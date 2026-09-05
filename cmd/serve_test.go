package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadServerFileConfig(t *testing.T) {
	t.Run("missing file returns empty config", func(t *testing.T) {
		cfg, err := loadServerFileConfig(t.TempDir())
		require.NoError(t, err)
		require.Equal(t, &serverFileConfig{}, cfg)
	})

	t.Run("parse config file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "config.json"),
			[]byte(`{"listen":"0.0.0.0","port":9000,"certificate_file":"certs/a.crt","certificate_key_file":"certs/a.key"}`),
			0600,
		))
		cfg, err := loadServerFileConfig(dir)
		require.NoError(t, err)
		require.Equal(t, "0.0.0.0", cfg.Listen)
		require.Equal(t, 9000, cfg.Port)
		require.Equal(t, "certs/a.crt", cfg.CertificateFile)
		require.Equal(t, "certs/a.key", cfg.CertificateKeyFile)
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{bad`), 0600))
		_, err := loadServerFileConfig(dir)
		require.Error(t, err)
	})
}

// buildServeConfig 依次套用默认值 -> 文件 -> CLI，返回最终配置
func buildServeConfig(dir string, file *serverFileConfig, cli *options) serveConfig {
	cfg := defaultServeConfig()
	cfg.overrideFromFile(file, dir)
	cfg.overrideFromCLI(cli)
	return cfg
}

func TestResolutionOrder(t *testing.T) {
	file := &serverFileConfig{
		Listen: "0.0.0.0", Port: 9000,
		CertificateFile:    "certs/a.crt",
		CertificateKeyFile: "certs/a.key",
	}
	cli := &options{
		host: "127.0.0.1", port: 9112,
		certificateFile: "/cli/x.crt", certificateKeyFile: "/cli/x.key",
	}
	// CLI 最终覆盖文件与默认值
	cfg := buildServeConfig("/wd", file, cli)
	require.Equal(t, "127.0.0.1", cfg.listen)
	require.Equal(t, 9112, cfg.port)
	require.Equal(t, "/cli/x.crt", cfg.certFile)
	require.Equal(t, "/cli/x.key", cfg.keyFile)

	// 无 CLI：文件覆盖默认值，证书相对 working_dir 解析
	cfg = buildServeConfig("/wd", file, &options{})
	require.Equal(t, "0.0.0.0", cfg.listen)
	require.Equal(t, 9000, cfg.port)
	require.Equal(t, filepath.Join("/wd", "certs/a.crt"), cfg.certFile)
	require.Equal(t, filepath.Join("/wd", "certs/a.key"), cfg.keyFile)

	// CLI 只覆盖部分字段：未提供的字段保留文件值
	cfg = buildServeConfig("/wd", file, &options{port: 8888})
	require.Equal(t, "0.0.0.0", cfg.listen)
	require.Equal(t, 8888, cfg.port)
	require.Equal(t, filepath.Join("/wd", "certs/a.crt"), cfg.certFile)

	// 全部为空：走代码写死默认值
	cfg = buildServeConfig("/wd", nil, &options{})
	require.Equal(t, defaultListen, cfg.listen)
	require.Equal(t, defaultPort, cfg.port)
	require.Empty(t, cfg.certFile)
}

func TestDefaultPort(t *testing.T) {
	require.Equal(t, 9112, defaultServeConfig().port)
}

func TestResolvePath(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.Equal(t, "", resolvePath("/wd", ""))
	})
	t.Run("relative resolved against working dir", func(t *testing.T) {
		require.Equal(t, filepath.Join("/wd", "certs/a.crt"), resolvePath("/wd", "certs/a.crt"))
	})
	t.Run("absolute kept", func(t *testing.T) {
		require.Equal(t, "/abs/a.crt", resolvePath("/wd", "/abs/a.crt"))
	})
	t.Run("env expansion", func(t *testing.T) {
		t.Setenv("SBCTL_TEST_DIR", "/envdir")
		require.Equal(t, filepath.Join("/envdir", "a.key"), resolvePath("/wd", "$SBCTL_TEST_DIR/a.key"))
	})
}

func TestResolveCwdPath(t *testing.T) {
	require.Equal(t, "", resolveCwdPath(""))
	require.Equal(t, "/abs/a.crt", resolveCwdPath("/abs/a.crt"))
	// 相对路径基于当前执行目录
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(wd, "a.crt"), resolveCwdPath("a.crt"))
}

func TestServeConfigValidate(t *testing.T) {
	t.Run("cert and key must be paired", func(t *testing.T) {
		cfg := serveConfig{listen: "127.0.0.1", port: 8080, certFile: "/x.crt"}
		require.Error(t, cfg.validate())
	})

	t.Run("no certs is valid", func(t *testing.T) {
		cfg := serveConfig{listen: "127.0.0.1", port: 8080}
		require.NoError(t, cfg.validate())
	})

	t.Run("missing file errors", func(t *testing.T) {
		dir := t.TempDir()
		cfg := serveConfig{
			listen: "127.0.0.1", port: 8080,
			certFile: filepath.Join(dir, "nope.crt"),
			keyFile:  filepath.Join(dir, "nope.key"),
		}
		require.Error(t, cfg.validate())
	})

	t.Run("both files exist is valid", func(t *testing.T) {
		dir := t.TempDir()
		crt := filepath.Join(dir, "a.crt")
		key := filepath.Join(dir, "a.key")
		require.NoError(t, os.WriteFile(crt, []byte("c"), 0600))
		require.NoError(t, os.WriteFile(key, []byte("k"), 0600))
		cfg := serveConfig{listen: "127.0.0.1", port: 8080, certFile: crt, keyFile: key}
		require.NoError(t, cfg.validate())
	})
}
