package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/spf13/cobra"
)

var (
	configFlagRestoreTemplate bool
	configFlagRestoreConfig   bool
	configFlagShowConfigHome  bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config commands",
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewCliLogger()
		cm := config.CreateConfManager()
		if configFlagRestoreTemplate {
			if err := cm.RestoreDefaultTemplate(); err != nil {
				log.Error(err)
			}
			return
		}

		if configFlagRestoreConfig {
			if err := cm.RestoreDefaultConfig(); err != nil {
				log.Error(err)
			}
			return
		}
		if configFlagShowConfigHome {
			log.Info(config.ConfigHome)
		}
	},
}

func init() {
	configCmd.Flags().BoolVarP(&configFlagRestoreTemplate, "restore-template", "T", false, "restore default template")
	configCmd.Flags().BoolVarP(&configFlagRestoreConfig, "restore-config", "C", false, "restore default config")
	configCmd.Flags().BoolVarP(&configFlagShowConfigHome, "show", "s", false, "show config home")
	rootCmd.AddCommand(configCmd)
}
