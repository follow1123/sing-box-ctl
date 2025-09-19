package cmd

import (
	"os"

	"github.com/follow1123/sing-box-ctl/config"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "list provider config",
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
		var tableData [][]string
		providers, err := provider.List()
		if err != nil {
			return err
		}
		if len(providers) == 0 {
			cmd.Println("no provider use 'provider add' subcommand to add")
			return nil
		}
		defaultProvider, err := provider.GetDefault()
		if err != nil {
			return err
		}
		for _, p := range providers {
			if p.Name == defaultProvider.Name {
				tableData = append(tableData, []string{p.Name, p.Url, "*"})
			} else {
				tableData = append(tableData, []string{p.Name, p.Url})
			}
		}
		table := tablewriter.NewTable(os.Stdout, tablewriter.WithEastAsian(false))
		table.Header("name", "url", "default")
		if err := table.Bulk(tableData); err != nil {
			return err
		}
		if err := table.Render(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
}
