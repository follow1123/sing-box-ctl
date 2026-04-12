package cmd

import (
	"errors"
	"fmt"
	"net"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/platform"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"

	"github.com/spf13/cobra"
)

var webuiCmd = &cobra.Command{
	Use:          "webui",
	Short:        "Open web UI with default browser",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		sb, err := settings.LoadConfigFromPath(conf.SingBoxConfigPath())
		if err != nil {
			return err
		}

		if sb.Experimental.ClashAPI == nil {
			return errors.New("web ui is not enabled")
		}
		addr := sb.Experimental.ClashAPI.ExternalController
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
