package cmd

import (
	_ "embed"
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/platform"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
	"github.com/spf13/cobra"
)

var (
	rcmenuFlagInit   bool
	rcmenuFlagDelete bool
)

const (
	menuStart   = "start"
	menuStop    = "stop"
	menuRestart = "restart"

	menuOpenWebUI    = "webui.open"
	menuDisableWebUI = "webui.disable"

	menuEnableMixedMode      = "mode.mixed"
	menuResetMixedMode       = "mode.mixed.reset"
	menuMixedEnableSysProxy  = "mode.mixed.sys_proxy.enable"
	menuMixedDisableSysProxy = "mode.mixed.sys_proxy.disable"
	menuMixedAllowLAN        = "mode.mixed.allow_lan"
	menuMixedDenyLAN         = "mode.mixed.deny_lan"

	menuEnableTunMode = "mode.tun"
)

//go:embed right_click_menu.reg
var menuReg []byte

var rcmenuCmd = &cobra.Command{
	Use:   "rcmenu [menu_name]",
	Short: "Integrate Windows right-click menu",
	Long: fmt.Sprintf(`Arguments:
[menu_name] string	menu name
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s
	- %s`,
		menuStart,
		menuStop,
		menuRestart,
		menuOpenWebUI,
		menuDisableWebUI,
		menuEnableMixedMode,
		menuResetMixedMode,
		menuMixedEnableSysProxy,
		menuMixedDisableSysProxy,
		menuMixedAllowLAN,
		menuMixedDenyLAN,
		menuEnableTunMode,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if rcmenuFlagInit {
			return initMenu()
		}
		if rcmenuFlagDelete {
			return deleteMenu()
		}
		if len(args) > 0 {
			conf, err := config.Default()
			if err != nil {
				return err
			}
			serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
			if err != nil {
				return err
			}
			updater, err := U.New(conf.SingBoxConfigPath())
			if err != nil {
				return err
			}
			isRunning := serv.IsRunning()
			menuName := args[0]
			switch menuName {
			case menuStart:
				err = serv.Start()
			case menuStop:
				err = serv.Stop()
			case menuRestart:
				err = serv.Restart()
			case menuOpenWebUI:
				act := U.NewWebUIStatusAction()
				isWebUIEnabled := act.IsEnabled(updater.JsonHandler())
				if isWebUIEnabled {
					err = serv.Start()
				} else {
					act.SetValue(true)
					err = updateAndRestart(isRunning, act, updater, serv)
				}
				addr, err := U.NewWebUIAddressAction().GetAddress(updater.JsonHandler())
				if err != nil {
					return err
				}
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					return fmt.Errorf("invalid address '%s'\n\t%w", addr, err)
				}
				platform.OpenUrl(fmt.Sprintf("http://localhost:%s", port))
			case menuDisableWebUI:
				act := U.NewWebUIStatusAction()
				act.SetValue(false)
				err = updateAndRestart(isRunning, act, updater, serv)
			case menuEnableMixedMode:
				isEnabled, err := U.NewMixedModeAction().IsEnabled(updater.JsonHandler())
				if err != nil {
					return err
				}
				if isEnabled {
					return nil
				}
				err = updateAndRestart(isRunning, U.NewMixedModeAction(), updater, serv)
			case menuResetMixedMode:
				err = updateAndRestart(isRunning, U.NewMixedModeAction(), updater, serv)
			case menuMixedEnableSysProxy, menuMixedDisableSysProxy, menuMixedAllowLAN, menuMixedDenyLAN:
				if err := ensureMixedModeEnabled(updater.JsonHandler()); err != nil {
					return err
				}
				var act U.Action
				switch menuName {
				case menuMixedEnableSysProxy:
					act = U.NewMixedSysProxyAction()
					act.SetValue(true)
				case menuMixedDisableSysProxy:
					act = U.NewMixedSysProxyAction()
					act.SetValue(false)
				case menuMixedAllowLAN:
					act = U.NewMixedAllowLANAction()
					act.SetValue(true)
				case menuMixedDenyLAN:
					act = U.NewMixedAllowLANAction()
					act.SetValue(false)
				}
				err = updateAndRestart(isRunning, act, updater, serv)
			case menuEnableTunMode:
				err = updateAndRestart(isRunning, U.NewTunModeAction(), updater, serv)
			default:
				return fmt.Errorf("invalid menu name: %s", menuName)
			}
			if err != nil {
				return err
			}
		} else {
			cmd.Help()
		}
		return nil
	},
}

func init() {
	rcmenuCmd.Flags().BoolVarP(&rcmenuFlagInit, "init", "i", false, "init right-click menu")
	rcmenuCmd.Flags().BoolVarP(&rcmenuFlagDelete, "delete", "d", false, "delete right-click menu")
	rootCmd.AddCommand(rcmenuCmd)
}

func ensureMixedModeEnabled(jh *jsonhandler.JsonHandler) error {
	mixedModeAct := U.NewMixedModeAction()
	isMixedModeEnabled, err := mixedModeAct.IsEnabled(jh)
	if err != nil {
		return err
	}
	if !isMixedModeEnabled {
		return fmt.Errorf("mixed mode is not enabled")
	}
	return nil
}

func updateAndRestart(isRunning bool, action U.Action, updater *U.Updater, serv service.Service) error {
	if err := updater.Update([]U.Action{action}, false); err != nil {
		return err
	}
	isModified := updater.IsModified()
	if isModified {
		if err := updater.Save(); err != nil {
			return err
		}
	}
	if isRunning && !isModified {
		return nil
	}
	return serv.Restart()
}

func initMenu() error {
	f, err := os.CreateTemp("", "sing-box-windows-right-click-menu-*.reg")
	if err != nil {
		return fmt.Errorf("create temp right-click reg file error:\n\t%w", err)
	}
	defer os.Remove(f.Name()) // 程序退出时自动删除
	if _, err := f.Write(menuReg); err != nil {
		return fmt.Errorf("write reg file error")
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp reg file error:\n\t%w", err)
	}

	command := fmt.Sprintf(`Start-Process -FilePath regedit -ArgumentList "/s", "%s" -WindowStyle Hidden -Verb RunAs`, f.Name())
	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "RemoteSigned", "-Command", command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}

func deleteMenu() error {
	command := `Start-Process -FilePath reg -ArgumentList "delete", "HKLM\SOFTWARE\Classes\DesktopBackground\Shell\SingBox", "/f" -WindowStyle Hidden -Verb RunAs`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "RemoteSigned", "-Command", command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}
