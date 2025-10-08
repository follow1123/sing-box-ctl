package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/spf13/cobra"
)

var (
	providerAddFlagSetDefault bool
)

var providerAddCmd = &cobra.Command{
	Use:          "add [flags] name url",
	Short:        "Add provider",
	Args:         cobra.ExactArgs(2),
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
		url := args[1]
		if err := provider.Add(name, url); err != nil {
			return err
		}
		if providerAddFlagSetDefault {
			if err := provider.SetDefault(name); err != nil {
				return err
			}
		}
		if err := provider.Save(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	providerAddCmd.Flags().BoolVarP(&providerAddFlagSetDefault, "set-default", "d", false, "set this provider to the default")

	providerCmd.AddCommand(providerAddCmd)
}
