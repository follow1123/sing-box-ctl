package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/platform"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:          "log",
	Short:        "Print log",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 初始化配置
		conf, err := config.Default()
		if err != nil {
			return err
		}
		sb, err := settings.LoadConfigFromPath(conf.SingBoxConfigPath())
		if err != nil {
			return fmt.Errorf("load config check log path error:\n\t%w", err)
		}

		if sb.Log == nil {
			return fmt.Errorf("no log config")
		}

		if sb.Log.Output == "" {
			return fmt.Errorf("no log file")
		}

		logFile := filepath.Join(conf.SingBoxWorkingDir(), sb.Log.Output)
		fmt.Printf("sing-box log file: %s\n\n", logFile)
		if err := platform.LogFile(logFile); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
