package cmd

import (
	"github.com/follow1123/sing-box-ctl/clash"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/services"
	"github.com/follow1123/sing-box-ctl/singbox"
	"github.com/spf13/cobra"
)

var (
	updateFlagWebui  string
	updateFlagMode   string
	updateFlagWrite  bool
	updateFlagRemote bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update local sing-box config",
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewCliLogger()
		cm := config.CreateConfManager()
		conf, err := cm.Load()
		if err != nil {
			log.Error(err)
			return
		}

		changeOpts := make(map[string]any)
		opts := make([]singbox.Opt, 0)
		if updateFlagWebui != "" {
			switch updateFlagWebui {
			case "true":
				opts = append(opts, singbox.WithWebUI(true))
				changeOpts["$.singbox.options.webui_enabled"] = true
			case "false":
				opts = append(opts, singbox.WithWebUI(false))
				changeOpts["$.singbox.options.webui_enabled"] = false
			default:
				opts = append(opts, singbox.WithWebUIAddr(updateFlagWebui))
				changeOpts["$.singbox.options.webui_enabled"] = true
				changeOpts["$.singbox.options.webui_addr"] = updateFlagWebui
			}
		}
		if updateFlagMode != "" {
			proxyMode, err := singbox.StrToProxyMode(updateFlagMode)
			if err != nil {
				log.Error(err)
				return
			}
			opts = append(opts, singbox.WithMode(proxyMode))
			changeOpts["$.singbox.options.mode"] = updateFlagMode
		}
		renderer, err := singbox.NewConfRenderer(log, cm.GetTemplateFile())
		if err != nil {
			log.Error(err)
			return
		}

		sb := singbox.NewSingBox(log, renderer, conf.SingBox)
		cu, err := clash.NewClashUpdater(log, cm.GetConfigHome(), conf.Services.Updater.Url, conf.Services.Updater.Interval)
		if err != nil {
			log.Error(err)
			return
		}

		updater, err := services.NewUpdater(log, sb, cu)
		if err != nil {
			log.Error(err)
			return
		}

		if err := updater.Update(updateFlagRemote, opts...); err != nil {
			log.Error(err)
			return
		}

		if updateFlagWrite {
			cm.ChangeConfig(changeOpts)
		}

	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateFlagWebui, "webui", "w", "", "enable or disalbe webui, available value: [true|false|<address>]")
	updateCmd.Flags().StringVarP(&updateFlagMode, "mode", "m", "", "switch mode, available value: [mixed|mixed-sp|tun]")
	updateCmd.Flags().BoolVarP(&updateFlagWrite, "write", "W", false, "write webui and mode flags to local configuration")
	updateCmd.Flags().BoolVarP(&updateFlagRemote, "remote", "r", false, "update remote config")
	rootCmd.AddCommand(updateCmd)
}
