package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
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

		table := tablewriter.NewTable(os.Stdout, tablewriter.WithEastAsian(false))
		jh, err := jsonhandler.FromFile(conf.SingBoxConfigPath())
		if err != nil {
			return err
		}
		tableData := [][]string{
			{"Status", status(serv.IsRunning())},
		}
		webUIStatusAct := U.NewWebUIStatusAction()
		if webUIStatusAct.IsEnabled(jh) {
			webUIAddrAct := U.NewWebUIAddressAction()
			addr, err := webUIAddrAct.GetAddress(jh)
			if err != nil {
				return err
			}
			tableData = append(tableData, []string{"WebUI Address", addr})
			webUISecretAct := U.NewWebUISecretAction()
			secret, err := webUISecretAct.GetSecret(jh)
			if err != nil {
				return err
			}
			if secret == "" {
				secret = "<not set>"
			}
			tableData = append(tableData, []string{"WebUI Secret", secret})
		} else {
			tableData = append(tableData, []string{"WebUI", switchStr(false)})
		}

		inboundType, exists := jh.GetString("inbounds.0.type")
		if !exists {
			return errors.New("no inbound or no inbound type")
		}
		switch inboundType {
		case "mixed":
			tableData = append(tableData, []string{"Mode", "mixed"})
			mixedPortAct := U.NewMixedPortAction()
			port, err := mixedPortAct.GetPort(jh)
			if err != nil {
				return err
			}
			tableData = append(tableData, []string{"Mixed Port", fmt.Sprintf("%d", port)})
			mixedSysProxyAct := U.NewMixedSysProxyAction()
			isSysProxyEnabled, err := mixedSysProxyAct.IsSysProxyEnabled(jh)
			if err != nil {
				return err
			}
			tableData = append(tableData, []string{"Mixed System Proxy", switchStr(isSysProxyEnabled)})
			mixedAllowLANAct := U.NewMixedAllowLANAction()
			isAllowLAN, err := mixedAllowLANAct.IsAllowLAN(jh)
			if err != nil {
				return err
			}
			tableData = append(tableData, []string{"Mixed Allow LAN", switchStr(isAllowLAN)})
		case "tun":
			tableData = append(tableData, []string{"Mode", "tun"})
		default:
			return fmt.Errorf("unsupported inbound type '%s'", inboundType)
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
