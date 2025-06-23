package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

//go:embed config.json.tmpl
var singBoxConfigTemplate []byte

type ConfigManager struct {
	confHome     string
	confFile     string
	templateFile string
}

func CreateConfManager() *ConfigManager {
	return NewConfManager(ConfigHome)
}

func NewConfManager(configHome string) *ConfigManager {
	expandHomePath := os.ExpandEnv(configHome)
	confHome := filepath.Clean(expandHomePath)
	cm := &ConfigManager{
		confHome:     confHome,
		confFile:     filepath.Join(confHome, "config.yaml"),
		templateFile: filepath.Join(confHome, "sing-box-config.json.tmpl"),
	}
	cm.init()
	return cm
}

func (cm *ConfigManager) init() {
	des, err := os.ReadDir(cm.confHome)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(cm.confHome, 0755); err != nil {
			fmt.Fprint(os.Stderr, fmt.Errorf("create config home %q error: \n\t%w\n", cm.confHome, err))
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Fprint(os.Stderr, fmt.Errorf("check config home error: \n\t%w\n", err))
		os.Exit(1)
	}

	if len(des) == 0 {
		if err := cm.RestoreDefaultConfig(); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		if err := cm.RestoreDefaultTemplate(); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func (cm *ConfigManager) Load() (*Config, error) {
	conf := defalutConf()
	data, err := os.ReadFile(cm.confFile)
	if err != nil {
		return nil, fmt.Errorf("load config error \n\t%w", err)
	}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, fmt.Errorf("parse config error \n\t%w", err)
	}
	return conf, nil
}

func (cm *ConfigManager) RestoreDefaultConfig() error {
	data, err := yaml.Marshal(defalutConf())
	if err != nil {
		return fmt.Errorf("load default config error: %w", err)
	}

	if err := os.WriteFile(cm.confFile, data, 0664); err != nil {
		return fmt.Errorf("write default config error: %w", err)
	}
	return nil
}

func (cm *ConfigManager) RestoreDefaultTemplate() error {
	if err := os.WriteFile(cm.templateFile, singBoxConfigTemplate, 0664); err != nil {
		return fmt.Errorf("write default template error: %w", err)
	}
	return nil
}

func (cm *ConfigManager) ChangeConfig(pathMap map[string]any) error {
	data, err := os.ReadFile(cm.confFile)
	if err != nil {
		return fmt.Errorf("read config error: \n\t%w", err)
	}

	confAst, err := parser.ParseBytes(data, 1)
	if err != nil {
		return fmt.Errorf("parse config error: \n\t%w", err)
	}

	for key, val := range pathMap {
		path, err := yaml.PathString(key)
		if err != nil {
			return fmt.Errorf("create path: %q error: \n\t%w", key, err)
		}
		exists, err := pathExists(confAst, path)
		if err != nil {
			return err
		}
		if exists {
			if err := path.ReplaceWithReader(confAst, strings.NewReader(fmt.Sprintf("%v", val))); err != nil {
				return fmt.Errorf("replace config %q error: \n\t%w", key, err)
			}
		} else {

			parentPath, err := parentPath(confAst, key)
			if err != nil {
				return err
			}
			if parentPath == nil {
				newAst, err := parser.ParseBytes([]byte(getMergeValue(key, val)), 1)
				if err != nil {
					return fmt.Errorf("create new ast error: \n\t%w", err)
				}
				confAst = newAst
			} else {
				k := fmt.Sprintf("$%s", key[len(parentPath.String()):])
				if err := parentPath.MergeFromReader(confAst, strings.NewReader(getMergeValue(k, val))); err != nil {
					return fmt.Errorf("merge config %q error: \n\t%w", key, err)
				}
			}
		}
	}

	if err := os.WriteFile(cm.confFile, []byte(confAst.String()), 0660); err != nil {
		return fmt.Errorf("save config error: \n\t%w", err)
	}
	return nil

}

func (cm *ConfigManager) GetConfigHome() string {
	return cm.confHome
}

func (cm *ConfigManager) GetConfigFile() string {
	return cm.confFile
}

func (cm *ConfigManager) GetTemplateFile() string {
	return cm.templateFile
}

func pathExists(astFile *ast.File, path *yaml.Path) (bool, error) {
	_, err := path.ReadNode(astFile)
	if errors.Is(err, yaml.ErrNotFoundNode) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("check path %q error: \n\t%w", path.String(), err)
	}
	return true, nil
}

func parentPath(astFile *ast.File, pathStr string) (*yaml.Path, error) {
	for {
		idx := strings.LastIndexByte(pathStr, '.')
		if idx == -1 {
			return nil, nil
		}
		pathStr = pathStr[0:idx]
		path, err := yaml.PathString(pathStr)
		if err != nil {
			return nil, fmt.Errorf("create path: %q error: \n\t%w", pathStr, err)
		}
		exists, err := pathExists(astFile, path)
		if err != nil {
			return nil, err
		}
		if exists {
			return path, nil
		}
	}
}

func getMergeValue(pathStr string, value any) string {
	keys := strings.Split(pathStr, ".")[1:]
	var sb strings.Builder
	for i, k := range keys {
		if i != 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(strings.Repeat("  ", i))
		sb.WriteString(k)
		sb.WriteString(":")
	}
	return fmt.Sprintf("%s %v", sb.String(), value)
}
