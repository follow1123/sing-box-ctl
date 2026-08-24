package cmd

import (
	"strconv"

	"github.com/follow1123/sing-box-ctl/config"
	S "github.com/follow1123/sing-box-ctl/settings"
	"github.com/spf13/cobra"
)

var (
	updateFlagEnableWebui  bool
	updateFlagDisableWebui bool
	updateFlagWebuiPort    uint16
	updateFlagWebuiSecret  string

	updateFlagEnableMixed              bool
	updateFlagDisableMixed             bool
	updateFlagMixedPort                uint16
	updateFlagMixedEnableSystemProxy   bool
	updateFlagMixedDisableSystemProxy  bool
	updateFlagMixedEnableProxySharing  bool
	updateFlagMixedDisableProxySharing bool

	updateFlagEnableTun  bool
	updateFlagDisableTun bool

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
		s, err := S.NewSettings(conf.SingBox.ConfigFile)
		if err != nil {
			return err
		}
		if err := s.SetTemplateConfig(conf.SingBoxTemplateConfigFile); err != nil {
			return err
		}

		settingsMap := make(map[S.SettingName]string)

		// clash_api 相关配置
		if updateFlagEnableWebui {
			settingsMap[S.StWebuiStatus] = "true"
		}
		if updateFlagWebuiPort != 0 {
			settingsMap[S.StWebuiPort] = strconv.FormatUint(uint64(updateFlagWebuiPort), 10)
		}
		if updateFlagWebuiSecret != "" {
			settingsMap[S.StWebuiSecret] = updateFlagWebuiSecret
		}
		if updateFlagDisableWebui {
			settingsMap[S.StWebuiStatus] = "false"
		}

		// inbound 模式相关配置
		if updateFlagEnableMixed {
			settingsMap[S.StMixedStatus] = "true"
		}
		if updateFlagMixedEnableSystemProxy {
			settingsMap[S.StMixedSysProxyStatus] = "true"
		}
		if updateFlagMixedEnableProxySharing {
			settingsMap[S.StMixedShareStatus] = "true"
		}
		if updateFlagMixedPort != 0 {
			settingsMap[S.StMixedPort] = strconv.FormatUint(uint64(updateFlagMixedPort), 10)
		}
		if updateFlagMixedDisableProxySharing {
			settingsMap[S.StMixedShareStatus] = "false"
		}
		if updateFlagMixedEnableSystemProxy {
			settingsMap[S.StMixedSysProxyStatus] = "false"
		}
		if updateFlagDisableMixed {
			settingsMap[S.StMixedStatus] = "false"
		}

		if updateFlagEnableTun {
			settingsMap[S.StTunStatus] = "true"
		}
		if updateFlagDisableTun {
			settingsMap[S.StTunStatus] = "false"
		}

		if err := s.SetMap(settingsMap); err != nil {
			return err
		}

		if err := s.SetPlatform(S.PlatformWindows); err != nil {
			return err
		}

		// 修改配置
		if err := s.Save(updateFlagFormat); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateFlagEnableWebui, "enable-webui", "w", false, "enable webui")
	updateCmd.Flags().BoolVarP(&updateFlagDisableWebui, "disable-webui", "W", false, "disable webui")
	updateCmd.Flags().Uint16Var(&updateFlagWebuiPort, "webui-port", 0, "webui address")
	updateCmd.Flags().StringVar(&updateFlagWebuiSecret, "webui-secret", "", "webui secret")

	updateCmd.Flags().BoolVarP(&updateFlagEnableMixed, "enable-mixed", "m", false, "enable mixed mode")
	updateCmd.Flags().BoolVarP(&updateFlagDisableMixed, "disable-mixed", "M", false, "disable mixed mode")
	updateCmd.Flags().Uint16Var(&updateFlagMixedPort, "mixed-port", 0, "mixed mode port")
	updateCmd.Flags().BoolVar(&updateFlagMixedEnableSystemProxy, "enable-sys-proxy", false, "enable system proxy in mixed mode")
	updateCmd.Flags().BoolVar(&updateFlagMixedDisableSystemProxy, "disable-sys-proxy", false, "disable system proxy in mixed mode")
	updateCmd.Flags().BoolVar(&updateFlagMixedEnableProxySharing, "enable-proxy-sharing", false, "enable proxy sharing")
	updateCmd.Flags().BoolVar(&updateFlagMixedDisableProxySharing, "disable-proxy-sharing", false, "disable proxy sharing")

	updateCmd.Flags().BoolVarP(&updateFlagEnableTun, "enable-tun", "t", false, "enable tun mode")
	updateCmd.Flags().BoolVarP(&updateFlagDisableTun, "disable-tun", "T", false, "disable tun mode")

	updateCmd.Flags().BoolVarP(&updateFlagFormat, "format", "f", false, "format config")
	rootCmd.AddCommand(updateCmd)
}
