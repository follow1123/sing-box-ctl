package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	S "github.com/follow1123/sing-box-ctl/settings"
	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconData []byte

type SingBoxTray struct {
	conf        *config.Config
	sts         *settings.Settings
	serv        *service.SingBoxService
	logFilePath string
}

func New() (*SingBoxTray, error) {
	conf, err := config.Default()
	if err != nil {
		return nil, err
	}

	sts, err := settings.NewSettings(conf.SingBox.ConfigFile)
	if err != nil {
		return nil, err
	}
	return &SingBoxTray{
		conf:        conf,
		sts:         sts,
		serv:        service.New(conf, sts),
		logFilePath: filepath.Join(conf.Home, "tray.log"),
	}, nil
}

func (s *SingBoxTray) Run() error {
	if s.serv.IsRunning() {
		return fmt.Errorf("sing-box is already running")
	}
	systray.Run(s.onReady, s.onExit)
	return nil
}

func (s *SingBoxTray) onReady() {
	s.Log("tray is ready.........")
	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("Awesome App")
	systray.SetTooltip("Sing Box Runner")

	mMixed := systray.AddMenuItem("混合代理模式", "混合代理模式")
	subMenuMixedStatus := mMixed.AddSubMenuItem("启用", "启用")
	subMenuMixedSysProxy := mMixed.AddSubMenuItemCheckbox("系统代理", "系统代理", false)
	subMenuMixedProxySharing := mMixed.AddSubMenuItemCheckbox("代理共享", "代理共享", false)

	mTun := systray.AddMenuItemCheckbox("Tun 模式", "Tun 模式", false)

	mWebui := systray.AddMenuItem("Web UI", "Web UI")
	subMenuWebuiStatus := mWebui.AddSubMenuItem("启用", "启用")
	subMenuWebuiOpen := mWebui.AddSubMenuItem("打开", "打开")

	mQuit := systray.AddMenuItem("退出", "Quit the whole app")

	if err := s.serv.Start(); err != nil {
		s.Fatal(err.Error())
	}
	s.Log("sing-box is running")

	for {
		if err := s.sts.Update(); err != nil {
			s.Fatal(err.Error())
		}

		if err := s.sts.SetTemplateConfig(s.conf.SingBoxTemplateConfigFile); err != nil {
			s.Fatal(fmt.Errorf("set template config error:\n\t%w", err).Error())
		}
		err := s.updateMenu(
			subMenuMixedStatus,
			subMenuMixedSysProxy,
			subMenuMixedProxySharing,
			mTun,
			subMenuWebuiStatus,
		)
		if err != nil {
			s.Fatal(fmt.Errorf("update settings error:\n\t%w", err).Error())
		}
		select {
		case <-subMenuMixedStatus.ClickedCh:
			mixedStatus := toggleTitle(subMenuMixedStatus, "启用", "禁用")
			if err := s.sts.Set(S.StMixedStatus, strconv.FormatBool(mixedStatus)); err != nil {
				s.Fatal(err.Error())
			}
		case <-subMenuMixedSysProxy.ClickedCh:
			sysProxyEnabled := toggleChecked(subMenuMixedSysProxy)
			if err := s.sts.Set(S.StMixedSysProxyStatus, strconv.FormatBool(sysProxyEnabled)); err != nil {
				s.Fatal(err.Error())
			}
		case <-subMenuMixedProxySharing.ClickedCh:
			proxySharingEnabled := toggleChecked(subMenuMixedProxySharing)
			if err := s.sts.Set(S.StMixedShareStatus, strconv.FormatBool(proxySharingEnabled)); err != nil {
				s.Fatal(err.Error())
			}
		case <-mTun.ClickedCh:
			tunEnabled := toggleChecked(mTun)
			if err := s.sts.Set(S.StTunStatus, strconv.FormatBool(tunEnabled)); err != nil {
				s.Fatal(err.Error())
			}
		case <-subMenuWebuiStatus.ClickedCh:
			// todo fix 无法修改配置文件
			webuiStatus := toggleTitle(subMenuWebuiStatus, "启用", "禁用")
			if err := s.sts.Set(S.StWebuiStatus, strconv.FormatBool(webuiStatus)); err != nil {
				s.Fatal(err.Error())
			}
		case <-subMenuWebuiOpen.ClickedCh:
			clashApi := s.sts.GetConfig().Experimental.ClashAPI
			if clashApi != nil && clashApi.ExternalController != "" {
				openUrl(fmt.Sprintf("http://%s", clashApi.ExternalController))
			} else {
				s.Log("webui is not enabled")
			}
			continue
		case <-mQuit.ClickedCh:
			systray.Quit()
			s.Log("exit tray.")
			return
		}

		if err := s.sts.Save(false); err != nil {
			s.Fatal(fmt.Errorf("save config error:\n\t%w", err).Error())
		}

		if err := s.serv.Restart(); err != nil {
			s.Fatal(err.Error())
		}
		s.Log("sing-box is restarting")
	}
}

func (s *SingBoxTray) updateMenu(
	subMenuMixedStatus,
	subMenuMixedSysProxy,
	subMenuMixedProxySharing,
	mTun,
	subMenuWebuiStatus *systray.MenuItem,
) error {
	mixedStatus, err := s.sts.GetBool(S.StMixedStatus)
	if err != nil {
		return err
	}
	mixedStatusTitle := "启用"
	if mixedStatus {
		mixedStatusTitle = "禁用"
	}
	setTitle(subMenuMixedStatus, mixedStatusTitle)

	if mixedStatus {
		sysProxyStatus, err := s.sts.GetBool(S.StMixedSysProxyStatus)
		if err != nil {
			return err
		}
		if sysProxyStatus {
			subMenuMixedSysProxy.Check()
		} else {
			subMenuMixedSysProxy.Uncheck()
		}

		mixedShareStatus, err := s.sts.GetBool(S.StMixedShareStatus)
		if err != nil {
			return err
		}
		if mixedShareStatus {
			subMenuMixedProxySharing.Check()
		} else {
			subMenuMixedProxySharing.Uncheck()
		}
	}

	tunStatus, err := s.sts.GetBool(S.StTunStatus)
	if err != nil {
		return err
	}
	if tunStatus {
		mTun.Check()
	} else {
		mTun.Uncheck()
	}

	webuiStatus, err := s.sts.GetBool(S.StWebuiStatus)
	if err != nil {
		return err
	}

	webuiStatusTitle := "启用"
	if webuiStatus {
		webuiStatusTitle = "禁用"
	}
	setTitle(subMenuWebuiStatus, webuiStatusTitle)

	return nil
}

func (s *SingBoxTray) writeLog(level string, msg string) {
	f, _ := os.OpenFile(s.logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	defer f.Close()
	logTime := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(f, "%s %-5s %s\n", logTime, level, msg)
}

func (s *SingBoxTray) Fatal(msg string) {
	defer os.Exit(1)
	s.writeLog("FATAL", msg)
}

func (s *SingBoxTray) Log(msg string) {
	s.writeLog("INFO", msg)
}

func (s *SingBoxTray) onExit() {
	if err := s.serv.Stop(); err != nil {
		s.Fatal(err.Error())
	}
}

func setTitle(item *systray.MenuItem, title string) {
	item.SetTitle(title)
	item.SetTooltip(title)
}

func toggleTitle(item *systray.MenuItem, enabledTitle, disabledTitle string) bool {
	status := item.String()
	if strings.Contains(status, enabledTitle) {
		item.SetTitle(disabledTitle)
		item.SetTooltip(disabledTitle)
		return true
	} else {
		item.SetTitle(enabledTitle)
		item.SetTooltip(enabledTitle)
		return false
	}
}

func toggleChecked(item *systray.MenuItem) bool {
	if item.Checked() {
		item.Uncheck()
	} else {
		item.Check()
	}
	return item.Checked()
}

func openUrl(url string) error {
	cmd := exec.Command("explorer", url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}
