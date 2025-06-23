package config

type SingBoxConfig struct {
	ConfigFile     string         `yaml:"config_file"`
	CheckCommand   string         `yaml:"check_cmd"`
	RestartCommand string         `yaml:"restart_cmd"`
	Options        SingBoxOptions `yaml:"options"`
}

type SingBoxOptions struct {
	WebUIEnabled bool   `yaml:"webui_enabled"`
	WebUIAddr    string `yaml:"webui_addr"`
	Mode         string `yaml:"mode"`
}

type ServicesConfig struct {
	Generator GeneratorConfig `yaml:"generator"`
	Updater   UpdaterConfig   `yaml:"updater"`
}

type GeneratorConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

type UpdaterConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Url      string `yaml:"url"`
	Interval string `yaml:"interval"`
}

type Config struct {
	SingBox  SingBoxConfig  `yaml:"singbox"`
	Services ServicesConfig `yaml:"services"`
}
