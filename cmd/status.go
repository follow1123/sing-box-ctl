package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/service"
	S "github.com/follow1123/sing-box-ctl/settings"
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
		s, err := S.NewSettings(conf.SingBox.ConfigFile)
		if err != nil {
			return err
		}
		serv := service.New(conf, s)

		cmd.Printf("Status: %s\n", status(serv.IsRunning()))

		webuiStatus, err := s.GetBool(S.StWebuiStatus)
		if err != nil {
			return err
		}

		if webuiStatus {
			port, err := s.GetUint16(S.StWebuiPort)
			if err != nil {
				return err
			}

			cmd.Printf("WebUI Address: %s\n", fmt.Sprintf("http://127.0.0.1:%d", port))

			secret, err := s.GetString(S.StWebuiSecret)
			if err != nil {
				return err
			}

			if secret != "" {
				cmd.Printf("WebUI Secret: %s\n", secret)
			}
		} else {
			cmd.Printf("WebUI: %s\n", switchStr(false))
		}

		mixedStatus, err := s.GetBool(S.StMixedStatus)
		if err != nil {
			return err
		}
		if mixedStatus {
			cmd.Printf("Mode: mixed\n")
			port, err := s.GetUint16(S.StMixedPort)
			if err != nil {
				return err
			}
			cmd.Printf("\tMixed Port: %d\n", port)
			sysProxyStatus, err := s.GetBool(S.StMixedSysProxyStatus)
			if err != nil {
				return err
			}
			cmd.Printf("\tMixed System Proxy: %s\n", switchStr(sysProxyStatus))
			proxySharingStatus, err := s.GetBool(S.StMixedShareStatus)
			if err != nil {
				return err
			}
			cmd.Printf("\tMixed Proxy Sharing: %s\n", switchStr(proxySharingStatus))
		}

		tunStatus, err := s.GetBool(S.StTunStatus)
		if err != nil {
			return err
		}
		if tunStatus {
			cmd.Printf("Mode: tun\n")
		}

		if s.GetConfig().Log != nil && s.GetConfig().Log.Output != "" {
			logFile := filepath.Join(conf.SingBox.WorkingDir, s.GetConfig().Log.Output)
			cmd.Printf("Log File: %s\n", logFile)
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
