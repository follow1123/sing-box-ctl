package cmd

import (
	"fmt"
	"os"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Status of sing-box",
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

		sb, err := settings.LoadConfigFromPath(conf.SingBoxConfigPath())

		table := tablewriter.NewTable(os.Stdout, tablewriter.WithEastAsian(false))
		tableData := [][]string{
			{"Status", status(serv.IsRunning())},
		}

		if err != nil {
			return err
		}
		if sb.Experimental.ClashAPI != nil {
			tableData = append(tableData, []string{"WebUI Address", sb.Experimental.ClashAPI.ExternalController})
			secret := sb.Experimental.ClashAPI.Secret
			if secret == "" {
				secret = "<not set>"
			}
			tableData = append(tableData, []string{"WebUI Secret", secret})
		} else {
			tableData = append(tableData, []string{"WebUI", switchStr(false)})
		}

		for _, inbound := range sb.Inbounds {
			if inbound["type"] == "mixed" {
				tableData = append(tableData, []string{"Mode", "mixed"})
				tableData = append(tableData, []string{"Mixed Port", fmt.Sprintf("%v", inbound["listen_port"])})
				tableData = append(tableData, []string{"Mixed System Proxy", switchStr(inbound["set_system_proxy"].(bool))})
			}
			if inbound["type"] == "tun" {
				tableData = append(tableData, []string{"Mode", "tun"})
			}

		}

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
	rootCmd.AddCommand(statusCmd)
}

func status(b bool) string {
	if b {
		return "running"
	} else {
		return "stopped"
	}
}

func switchStr(b bool) string {
	if b {
		return "on"
	} else {
		return "off"
	}
}
