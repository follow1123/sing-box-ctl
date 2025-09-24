package cmd

import (
	"errors"
	"fmt"
	"net"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/platform"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
	"github.com/spf13/cobra"
)

var webuiCmd = &cobra.Command{
	Use:   "webui",
	Short: "Open web UI with default browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		// 初始化配置
		conf, err := config.Default()
		if err != nil {
			return err
		}
		serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
		if err != nil {
			return err
		}
		if !serv.IsRunning() {
			return errors.New("service is not running")
		}
		jh, err := jsonhandler.FromFile(conf.SingBoxConfigPath())
		if err != nil {
			return err
		}
		webUIStatusAct := U.NewWebUIStatusAction()
		if !webUIStatusAct.IsEnabled(jh) {
			return errors.New("web ui is not enabled")
		}
		webUIAddrAct := U.NewWebUIAddressAction()
		addr, err := webUIAddrAct.GetAddress(jh)
		if err != nil {
			return err
		}
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("invalid address '%s'\n\t%w", addr, err)
		}
		platform.OpenUrl(fmt.Sprintf("http://localhost:%s", port))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webuiCmd)
}
