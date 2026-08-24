package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/provider"
)

//go:embed singbox_template_config.json
var singboxTemplateConfigData []byte

const (
	// TemplateFileName 转换模板文件名（位于 working_dir 下）
	TemplateFileName = "singbox_template_config.json"
	// ProvidersDirName provider 订阅数据目录名（位于 working_dir 下）
	ProvidersDirName = "providers"
)

type Config struct {
	ConfigPath   string
	WorkingDir   string
	ProvidersDir string
	TemplateFile string
	Providers    []provider.ProviderConfig
}

// New 从主配置文件加载配置，解析路径并初始化工作目录
func New(configPath string) (*Config, error) {
	configPath = filepath.Clean(configPath)

	p, err := provider.New(configPath)
	if err != nil {
		return nil, err
	}
	workingDir := p.WorkingDir()
	if workingDir == "" {
		return nil, fmt.Errorf("working_dir is not set in config file %s", configPath)
	}

	providersDir := filepath.Join(workingDir, ProvidersDirName)
	templateFile := filepath.Join(workingDir, TemplateFileName)

	// 初始化工作目录
	if err := os.MkdirAll(providersDir, 0700); err != nil {
		return nil, fmt.Errorf("init providers dir error:\n\t%w", err)
	}

	// 初始化转换模板
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		if err := os.WriteFile(templateFile, singboxTemplateConfigData, 0600); err != nil {
			return nil, fmt.Errorf("init template config error:\n\t%w", err)
		}
	}

	return &Config{
		ConfigPath:   configPath,
		WorkingDir:   workingDir,
		ProvidersDir: providersDir,
		TemplateFile: templateFile,
		Providers:    p.List(),
	}, nil
}
