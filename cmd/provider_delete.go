package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/spf13/cobra"
)

var providerDeleteCmd = &cobra.Command{
	Use:          "delete [flags] name",
	Short:        "Delete provider by name",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true, // 关闭错误时的帮助信息
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Default()
		if err != nil {
			return err
		}
		provider, err := P.New(conf.ConfigPath())
		if err != nil {
			return err
		}
		name := args[0]
		if err := provider.Delete(name); err != nil {
			return err
		}
		if err := provider.Save(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	providerCmd.AddCommand(providerDeleteCmd)
}
