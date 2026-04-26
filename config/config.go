package config

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed singbox_template_config.json
var singboxTemplateConfigData []byte

type Config struct {
	Home                      string
	ConfigFile                string
	ArchiveDir                string
	SingBoxTemplateConfigFile string
	SingBoxServiceScript      string
	SingBox                   SingBoxConfig
}

type SingBoxConfig struct {
	Binary     string
	ConfigFile string
	WorkingDir string
}

func New(home string) (*Config, error) {
	home = filepath.Clean(os.ExpandEnv(home))

	singboxBin, err := exec.LookPath("sing-box")
	if err != nil {
		return nil, fmt.Errorf("cannot find sing-box find in path:\n\t%w", err)
	}
	singboxDir := filepath.Join(home, "singbox")
	singBoxConfig := SingBoxConfig{
		Binary:     singboxBin,
		ConfigFile: filepath.Join(singboxDir, "config.json"),
		WorkingDir: filepath.Join(singboxDir, "working_dir"),
	}

	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, fmt.Errorf("init config home '%s' error: \n\t%w", home, err)
	}
	if err := os.MkdirAll(singboxDir, 0700); err != nil {
		return nil, fmt.Errorf("init singbox home error:\n\t%w", err)
	}

	// 初始化 singbox 模板配置
	singboxTemplateConfigFile := filepath.Join(home, "singbox_template_config.json")
	if _, err := os.Stat(singboxTemplateConfigFile); os.IsNotExist(err) {
		if err := os.WriteFile(singboxTemplateConfigFile, singboxTemplateConfigData, 0600); err != nil {
			return nil, fmt.Errorf("init template config error:\n\t%w", err)
		}
	}

	// 初始化 singbox 脚本
	singboxScriptFile := filepath.Join(home, ServiceScript)
	if _, err := os.Stat(singboxScriptFile); os.IsNotExist(err) {
		if err := os.WriteFile(singboxScriptFile, singboxServiceScriptData, 0700); err != nil {
			return nil, fmt.Errorf("init singbox service script error:\n\t%w", err)
		}
	}

	return &Config{
		Home:                      home,
		ConfigFile:                filepath.Join(home, "config.json"),
		ArchiveDir:                filepath.Join(home, "archived"),
		SingBoxTemplateConfigFile: singboxTemplateConfigFile,
		SingBoxServiceScript:      singboxScriptFile,
		SingBox:                   singBoxConfig,
	}, nil
}

func Default() (*Config, error) {
	return New(ConfigHome)
}
