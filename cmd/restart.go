package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:          "restart",
	Short:        "Restart sing-box",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Default()
		if err != nil {
			return err
		}
		serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
		if err != nil {
			return err
		}
		if err := serv.Restart(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
