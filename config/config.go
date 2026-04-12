package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed config_tmpl.json
var singBoxConfigTemplate string

type Config struct {
	home                  string
	configPath            string
	singBoxBinaryPath     string
	singBoxConfigPath     string
	singBoxTmplConfigPath string
	singBoxWorkingDir     string
	archiveDir            string
}

func New(home string) (*Config, error) {
	home = filepath.Clean(os.ExpandEnv(home))

	if err := os.MkdirAll(home, 0755); err != nil {
		return nil, fmt.Errorf("init config home '%s' error: \n\t%w", home, err)
	}
	singBoxTmplConfigPath := filepath.Join(home, "config_tmpl.json")
	file, err := os.OpenFile(singBoxTmplConfigPath, os.O_CREATE|os.O_EXCL, 0660)
	var tmplExists bool
	if err != nil {
		if os.IsExist(err) {
			tmplExists = true
		} else {
			return nil, fmt.Errorf("open template config %s error:\n\t%w", singBoxTmplConfigPath, err)
		}
	}
	defer file.Close()
	if !tmplExists {
		_, err := io.WriteString(file, singBoxConfigTemplate)
		if err != nil {
			return nil, fmt.Errorf("init template config error:\n\t%w", err)
		}
	}

	return &Config{
		home:                  home,
		configPath:            filepath.Join(home, "sing-box-ctl-config.json"),
		singBoxBinaryPath:     filepath.Join(home, BinaryName),
		singBoxConfigPath:     filepath.Join(home, "config.json"),
		singBoxTmplConfigPath: singBoxTmplConfigPath,
		singBoxWorkingDir:     filepath.Join(home, "wd"),
		archiveDir:            filepath.Join(home, "archived_config"),
	}, nil
}

func Default() (*Config, error) {
	return New(ConfigHome)
}

func (c Config) Home() string {
	return c.home
}

func (c Config) ConfigPath() string {
	return c.configPath
}

func (c Config) SingBoxBinaryPath() string {
	return c.singBoxBinaryPath
}
func (c Config) SingBoxConfigPath() string {
	return c.singBoxConfigPath
}

func (c Config) SingBoxTmplConfigPath() string {
	return c.singBoxTmplConfigPath
}

func (c Config) SingBoxWorkingDir() string {
	return c.singBoxWorkingDir
}

func (c Config) ArchiveDir() string {
	return c.archiveDir
}
