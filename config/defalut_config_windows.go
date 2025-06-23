package config

var ConfigHome = "%LOCALAPPDATA%/sing-box-ctl"

func defalutConf() *Config {
	return &Config{
		SingBox: SingBoxConfig{
			ConfigFile: "<your-sing-box-config-file>",
			Options: SingBoxOptions{
				WebUIEnabled: true,
				WebUIAddr:    "127.0.0.1:9090",
				Mode:         "mixed",
			},
			CheckCommand:   "sing-box check -c %s",
			RestartCommand: "<your-restart-command>",
		},
		Services: ServicesConfig{
			Generator: GeneratorConfig{
				Enabled: true,
				Addr:    ":7789",
			},
			Updater: UpdaterConfig{
				Enabled:  true,
				Url:      "<your-subscription-url>",
				Interval: "1h",
			},
		},
	}
}
