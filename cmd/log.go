package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/jsonhandler"
	"github.com/follow1123/sing-box-ctl/platform"
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
		jh, err := jsonhandler.FromFile(conf.SingBoxConfigPath())
		if err != nil {
			return err
		}
		logName, exists := jh.GetString("log.output")
		if !exists {
			return fmt.Errorf("no log file")
		}
		logFile := filepath.Join(conf.SingBoxWorkingDir(), logName)
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
