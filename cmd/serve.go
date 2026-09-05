package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/webui"
)

// 代码写死的默认值；任一选项最终生效顺序：命令行 > config.json > 此处默认值
const (
	defaultListen = "127.0.0.1"
	defaultPort   = 9112
)

// serveConfig 生效后的监听与证书配置
type serveConfig struct {
	listen   string
	port     int
	certFile string
	keyFile  string
}

// defaultServeConfig 返回代码写死的默认配置（无证书 = http）
func defaultServeConfig() serveConfig {
	return serveConfig{listen: defaultListen, port: defaultPort}
}

// serverFileConfig 对应 working_dir/config.json（可选文件）。
// 其中证书相对路径以 working_dir 为基准解析。
type serverFileConfig struct {
	Port               int    `json:"port"`
	Listen             string `json:"listen"`
	CertificateFile    string `json:"certificate_file"`
	CertificateKeyFile string `json:"certificate_key_file"`
}

func serveCmd(workingDir string, cli *options) error {
	// 生效顺序：默认值 -> 配置文件 -> 命令行
	cfg := defaultServeConfig()

	fileCfg, err := loadServerFileConfig(workingDir)
	if err != nil {
		return err
	}
	cfg.overrideFromFile(fileCfg, workingDir)

	cfg.overrideFromCLI(cli)

	if err := cfg.validate(); err != nil {
		return err
	}
	server, err := webui.New(webui.Options{
		WorkingDir: workingDir,
		Listen:     cfg.listen,
		Port:       cfg.port,
		CertFile:   cfg.certFile,
		KeyFile:    cfg.keyFile,
	})
	if err != nil {
		return err
	}
	return server.Serve()
}

// loadServerFileConfig 读取工作目录下可选的 config.json；文件不存在时返回空配置
func loadServerFileConfig(workingDir string) (*serverFileConfig, error) {
	cfg := &serverFileConfig{}
	path := filepath.Join(workingDir, "config.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config file %s error:\n\t%w", path, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s error:\n\t%w", path, err)
	}
	return cfg, nil
}

// overrideFromFile 用配置文件覆盖当前配置（字段缺失或零值则跳过）。
// 证书两个字段只要出现任一即整体按配置文件的证书对处理。
func (c *serveConfig) overrideFromFile(file *serverFileConfig, workingDir string) {
	if file == nil {
		return
	}
	if file.Port > 0 {
		c.port = file.Port
	}
	if file.Listen != "" {
		c.listen = file.Listen
	}
	if file.CertificateFile != "" || file.CertificateKeyFile != "" {
		c.certFile = resolvePath(workingDir, file.CertificateFile)
		c.keyFile = resolvePath(workingDir, file.CertificateKeyFile)
	}
}

// overrideFromCLI 用命令行显式参数覆盖当前配置（未显式提供的字段跳过）。
// 证书相对路径按当前执行目录解析。
func (c *serveConfig) overrideFromCLI(cli *options) {
	if cli == nil {
		return
	}
	if cli.host != "" {
		c.listen = cli.host
	}
	if cli.port > 0 {
		c.port = cli.port
	}
	if cli.certificateFile != "" || cli.certificateKeyFile != "" {
		c.certFile = resolveCwdPath(cli.certificateFile)
		c.keyFile = resolveCwdPath(cli.certificateKeyFile)
	}
}

// validate 校验生效配置：证书与私钥需成对配置且文件存在
func (c serveConfig) validate() error {
	if (c.certFile == "") != (c.keyFile == "") {
		return fmt.Errorf("certificate file and key file must be configured as a pair")
	}
	if c.certFile == "" {
		return nil
	}
	for _, p := range []string{c.certFile, c.keyFile} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("certificate file not found: %s", p)
		}
	}
	return nil
}

// resolvePath 相对路径基于 working_dir 解析（用于 config.json）；支持环境变量与绝对路径
func resolvePath(workingDir, p string) string {
	if p == "" {
		return ""
	}
	p = os.ExpandEnv(p)
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(workingDir, p)
}

// resolveCwdPath 命令行传入的相对路径按当前执行目录解析；支持环境变量与绝对路径
func resolveCwdPath(p string) string {
	if p == "" {
		return ""
	}
	p = os.ExpandEnv(p)
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}
