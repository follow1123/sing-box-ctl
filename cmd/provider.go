package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:          "provider",
	Short:        "Manage provider config",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Default()
		if err != nil {
			return err
		}
		provider, err := P.New(conf.ConfigFile)
		if err != nil {
			return err
		}
		providers := provider.List()
		if len(providers) == 0 {
			cmd.Println("no provider use 'provider add' subcommand to add")
			return nil
		}
		defaultProvider := provider.GetDefault()
		for _, p := range providers {
			if p.Name == defaultProvider.Name {
				cmd.Printf("%s(default): %s\n", p.Name, p.Url)
			} else {
				cmd.Printf("%s: %s\n", p.Name, p.Url)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
}
