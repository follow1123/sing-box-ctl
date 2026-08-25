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
	// DefaultTemplateName 默认模板文件名（不含扩展名）
	DefaultTemplateName = "default"
	// TemplatesDirName 模板目录名（位于 working_dir 下）
	TemplatesDirName = "templates"
	// ProvidersDirName provider 订阅数据目录名（位于 working_dir 下）
	ProvidersDirName = "providers"
)

type Config struct {
	ConfigPath   string
	WorkingDir   string
	ProvidersDir string
	TemplatesDir string
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
	templatesDir := filepath.Join(workingDir, TemplatesDirName)

	// 初始化工作目录
	if err := os.MkdirAll(providersDir, 0700); err != nil {
		return nil, fmt.Errorf("init providers dir error:\n\t%w", err)
	}
	if err := os.MkdirAll(templatesDir, 0700); err != nil {
		return nil, fmt.Errorf("init templates dir error:\n\t%w", err)
	}

	// 模板目录为空时初始化默认模板
	if err := initDefaultTemplate(templatesDir); err != nil {
		return nil, err
	}

	return &Config{
		ConfigPath:   configPath,
		WorkingDir:   workingDir,
		ProvidersDir: providersDir,
		TemplatesDir: templatesDir,
		Providers:    p.List(),
	}, nil
}

// TemplateData 返回内嵌的默认模板内容
func TemplateData() []byte {
	return singboxTemplateConfigData
}

// initDefaultTemplate 模板目录为空时，从内嵌数据创建默认模板
func initDefaultTemplate(templatesDir string) error {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return fmt.Errorf("read templates dir error:\n\t%w", err)
	}
	if len(entries) > 0 {
		return nil
	}
	defaultFile := filepath.Join(templatesDir, DefaultTemplateName+".json")
	if err := os.WriteFile(defaultFile, singboxTemplateConfigData, 0600); err != nil {
		return fmt.Errorf("init default template error:\n\t%w", err)
	}
	return nil
}
