package cmd

import (
	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/spf13/cobra"
)

var providerDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete provider by name",
	Args:  cobra.ExactArgs(1),
	Long: `Arguments:
	<name> string 	provider name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
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
