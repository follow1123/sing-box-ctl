package cmd

import (
	"fmt"

	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/spf13/cobra"
)

var (
	providerUpdateFlagSetDefault bool
)

var providerUpdateCmd = &cobra.Command{
	Use:   "update <name> [url]",
	Short: "Update provider",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("'name' is required")
		}
		if len(args) > 2 {
			return fmt.Errorf("only two parameters 'name' and 'url'")
		}
		return nil
	},
	Long: `Arguments:
	<name> string 	provider name
	[url] string	provider url (optinoal)`,
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
		if len(args) > 1 {
			url := args[1]
			if err := provider.Update(name, url); err != nil {
				return err
			}
		}
		if providerUpdateFlagSetDefault {
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
	providerUpdateCmd.Flags().BoolVarP(&providerUpdateFlagSetDefault, "set-default", "d", false, "set this provider to the default")

	providerCmd.AddCommand(providerUpdateCmd)
}
