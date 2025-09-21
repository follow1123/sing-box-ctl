package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	home              string
	configPath        string
	singBoxBinaryPath string
	singBoxConfigPath string
	singBoxWorkingDir string
	archiveDir        string
}

func New(home string) (*Config, error) {
	home = filepath.Clean(os.ExpandEnv(home))

	if err := os.MkdirAll(home, 0755); err != nil {
		return nil, fmt.Errorf("init config home '%s' error: \n\t%w", home, err)
	}
	return &Config{
		home:              home,
		configPath:        filepath.Join(home, "sing-box-ctl-config.json"),
		singBoxBinaryPath: filepath.Join(home, BinaryName),
		singBoxConfigPath: filepath.Join(home, "config.json"),
		singBoxWorkingDir: filepath.Join(home, "wd"),
		archiveDir:        filepath.Join(home, "archived_config"),
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

func (c Config) SingBoxWorkingDir() string {
	return c.singBoxWorkingDir
}

func (c Config) ArchiveDir() string {
	return c.archiveDir
}
