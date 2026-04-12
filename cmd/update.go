package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/spf13/cobra"
)

var (
	updateFlagDisableWebui bool
	updateFlagResetWebui   bool
	updateFlagWebuiAddr    string
	updateFlagWebuiSecret  string

	updateFlagMixedMode               bool
	updateFlagMixedPort               uint16
	updateFlagMixedEnableSystemProxy  bool
	updateFlagMixedDisableSystemProxy bool
	updateFlagMixedAllowLAN           bool
	updateFlagMixedDenyLAN            bool
	updateFlagTunMode                 bool

	updateFlagRestart bool

	updateFlagFormat bool
)

var updateCmd = &cobra.Command{
	Use:          "update",
	Short:        "Modify the configuration file you specified",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 初始化配置
		conf, err := config.Default()
		if err != nil {
			return err
		}
		s, err := settings.NewSettings(conf.SingBoxTmplConfigPath(), conf.SingBoxConfigPath())
		if err != nil {
			return err
		}
		// clash_api 相关配置
		webuiSetting := settings.WebUISettings{Enabled: true}
		if cmd.Flags().Changed("webui-addr") {
			webuiSetting.Addr = &updateFlagWebuiAddr
		}
		if cmd.Flags().Changed("webui-secret") {
			webuiSetting.Password = &updateFlagWebuiSecret
		}
		if updateFlagResetWebui {
			webuiSetting.Reset = true
		}
		if updateFlagDisableWebui {
			webuiSetting.Enabled = false
		}
		s.UpdateWebUISettings(webuiSetting)

		// inbound 模式相关配置
		mixedProxySettings := settings.MixedProxySettings{Enabled: true}
		if updateFlagMixedPort != 0 {
			mixedProxySettings.Port = &updateFlagMixedPort
		}
		if updateFlagMixedEnableSystemProxy {
			mixedProxySettings.EnableSystemProxy = &updateFlagMixedEnableSystemProxy
		}
		if updateFlagMixedDisableSystemProxy {
			updateFlagMixedDisableSystemProxy = !updateFlagMixedDisableSystemProxy
			mixedProxySettings.EnableSystemProxy = &updateFlagMixedEnableSystemProxy
		}
		if updateFlagMixedAllowLAN {
			mixedProxySettings.AllowLAN = &updateFlagMixedAllowLAN
		}
		if updateFlagMixedDenyLAN {
			updateFlagMixedDenyLAN = !updateFlagMixedDenyLAN
			mixedProxySettings.AllowLAN = &updateFlagMixedAllowLAN
		}
		if updateFlagMixedMode {
			mixedProxySettings.Reset = true
		}
		if err := s.UpdateMixedProxySettings(mixedProxySettings); err != nil {
			return err
		}

		if updateFlagTunMode {
			s.UpdateTunSettings(true)
		}
		// 修改配置
		if err := s.Save(updateFlagFormat); err != nil {
			return err
		}

		// 重启服务
		if updateFlagRestart {
			serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
			if err != nil {
				return err
			}
			// 服务已启动，配置未修改，直接退出
			if serv.IsRunning() {
				return nil
			}
			if err := serv.Restart(); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateFlagResetWebui, "reset-webui", "w", false, "reset webui config")
	updateCmd.Flags().BoolVarP(&updateFlagDisableWebui, "disable-webui", "W", false, "disable webui")
	updateCmd.Flags().StringVar(&updateFlagWebuiAddr, "webui-addr", "", "webui address")
	updateCmd.Flags().StringVar(&updateFlagWebuiSecret, "webui-secret", "", "webui secret")

	updateCmd.Flags().BoolVarP(&updateFlagMixedMode, "mixed", "m", false, "reset to mixed mode")
	updateCmd.Flags().Uint16Var(&updateFlagMixedPort, "mixed-port", 0, "mixed mode port")
	updateCmd.Flags().BoolVarP(&updateFlagMixedEnableSystemProxy, "enable-sys-proxy", "s", false, "enable system proxy in mixed mode")
	updateCmd.Flags().BoolVarP(&updateFlagMixedDisableSystemProxy, "disable-sys-proxy", "S", false, "disable system proxy in mixed mode")
	updateCmd.Flags().BoolVarP(&updateFlagMixedAllowLAN, "allow-lan", "l", false, "allow LAN Sharing")
	updateCmd.Flags().BoolVarP(&updateFlagMixedDenyLAN, "deny-lan", "L", false, "deny LAN Sharing")
	updateCmd.Flags().BoolVarP(&updateFlagTunMode, "tun", "t", false, "reset to tun mode")

	updateCmd.Flags().BoolVarP(&updateFlagRestart, "restart", "r", false, "restart service")

	updateCmd.Flags().BoolVarP(&updateFlagFormat, "format", "f", false, "format config")
	rootCmd.AddCommand(updateCmd)
}
