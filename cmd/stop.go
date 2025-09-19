package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop sing-box",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		conf, err := config.Default()
		if err != nil {
			return err
		}
		serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
		if err != nil {
			return err
		}
		if err := serv.Stop(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
