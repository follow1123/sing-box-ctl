package cmd

import (
	"strings"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
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
	Use:   "update",
	Short: "Modify the configuration file you specified",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		// 初始化配置
		conf, err := config.Default()
		if err != nil {
			return err
		}
		// 初始化 updater
		updater, err := U.New(conf.SingBoxConfigPath())
		if err != nil {
			return err
		}
		var actions []U.Action
		// clash_api 相关配置
		if cmd.Flags().Changed("webui-addr") {
			act := U.NewWebUIAddressAction()
			act.SetValue(updateFlagWebuiAddr)
			actions = append(actions, act)
		}
		if cmd.Flags().Changed("webui-secret") {
			act := U.NewWebUISecretAction()
			act.SetValue(strings.TrimSpace(updateFlagWebuiSecret))
			actions = append(actions, act)
		}
		if updateFlagResetWebui {
			act := U.NewWebUIStatusAction()
			act.SetValue(true)
			actions = append(actions, act)
		}
		if updateFlagDisableWebui {
			act := U.NewWebUIStatusAction()
			act.SetValue(false)
			actions = append(actions, act)
		}
		// inbound 模式相关配置
		if updateFlagMixedPort != 0 {
			act := U.NewMixedPortAction()
			act.SetValue(updateFlagMixedPort)
			actions = append(actions, act)
		}
		if updateFlagMixedEnableSystemProxy {
			act := U.NewMixedSysProxyAction()
			act.SetValue(true)
			actions = append(actions, act)
		}
		if updateFlagMixedDisableSystemProxy {
			act := U.NewMixedSysProxyAction()
			act.SetValue(false)
			actions = append(actions, act)
		}
		if updateFlagMixedAllowLAN {
			act := U.NewMixedAllowLANAction()
			act.SetValue(true)
			actions = append(actions, act)
		}
		if updateFlagMixedDenyLAN {
			act := U.NewMixedAllowLANAction()
			act.SetValue(false)
			actions = append(actions, act)
		}
		if updateFlagMixedMode {
			actions = append(actions, U.NewMixedModeAction())
		}
		if updateFlagTunMode {
			actions = append(actions, U.NewTunModeAction())
		}
		// 修改配置
		if err := updater.Update(actions, updateFlagFormat); err != nil {
			return err
		}
		isModified := updater.IsModified()
		if isModified {
			if err := updater.Save(); err != nil {
				return err
			}
		}
		// 重启服务
		if updateFlagRestart {
			serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
			if err != nil {
				return err
			}
			// 服务已启动，配置未修改，直接退出
			if serv.IsRunning() && !isModified {
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
