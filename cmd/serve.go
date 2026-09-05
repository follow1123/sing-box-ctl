package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/webui"
)

const (
	defaultListen = "127.0.0.1"
	defaultPort   = 8080
)

// serverFileConfig 对应 working_dir/config.json（可选），命令行显式参数优先于文件
type serverFileConfig struct {
	Port               int    `json:"port"`
	Listen             string `json:"listen"`
	CertificateFile    string `json:"certificate_file"`
	CertificateKeyFile string `json:"certificate_key_file"`
}

// mergedConfig 合并 CLI / 配置文件 / 默认值后的最终配置
type mergedConfig struct {
	listen   string
	port     int
	certFile string
	keyFile  string
}

func serveCmd(workingDir, cliListen string, cliPort int) error {
	fileCfg, err := loadServerFileConfig(workingDir)
	if err != nil {
		return err
	}
	cfg := mergeServerConfig(cliListen, cliPort, fileCfg, workingDir)
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

// mergeServerConfig 合并优先级：命令行显式参数 > 配置文件 > 内置默认值。
// 证书路径为相对路径时基于工作目录解析。
func mergeServerConfig(cliListen string, cliPort int, file *serverFileConfig, workingDir string) mergedConfig {
	cfg := mergedConfig{listen: defaultListen, port: defaultPort}
	if file != nil {
		if file.Listen != "" {
			cfg.listen = file.Listen
		}
		if file.Port > 0 {
			cfg.port = file.Port
		}
		cfg.certFile = resolvePath(workingDir, file.CertificateFile)
		cfg.keyFile = resolvePath(workingDir, file.CertificateKeyFile)
	}
	if cliListen != "" {
		cfg.listen = cliListen
	}
	if cliPort > 0 {
		cfg.port = cliPort
	}
	return cfg
}

// validate 校验合并后的配置：证书需成对存在且文件可达
func (c mergedConfig) validate() error {
	if (c.certFile == "") != (c.keyFile == "") {
		return fmt.Errorf("certificate_file and certificate_key_file must be set together in config.json")
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

// resolvePath 相对路径基于工作目录解析；支持环境变量与绝对路径
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
