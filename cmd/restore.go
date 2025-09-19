package cmd

import (
	"errors"
	"fmt"
	"os"

	A "github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
	"github.com/spf13/cobra"
)

var (
	restoreFlagFormat  bool
	restoreFlagRestart bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore config from last provider archived config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		conf, err := config.Default()
		if err != nil {
			return err
		}
		// 获取最新的归档配置
		archiver, err := A.New(conf.ArchiveDir())
		if err != nil {
			return err
		}
		latestArchive := archiver.GetLatest()
		if latestArchive == "" {
			return errors.New("no latest archive")
		}
		data, err := os.ReadFile(latestArchive)
		if err != nil {
			return fmt.Errorf("read latest archive '%s' error:\n\t%w", latestArchive, err)
		}
		// 转换成 sing-box 配置
		newConfig, err := converter.Convert(data)
		if err != nil {
			return err
		}
		var finalConfig []byte
		singBoxConfigPath := conf.SingBoxConfigPath()
		updater, err := U.New(singBoxConfigPath)
		if err != nil {
			// 文件不存在直接作为最终的配置
			finalConfig = newConfig
		} else {
			if err := updater.Upgrade(newConfig, restoreFlagFormat); err != nil {
				return err
			}
			finalConfig = updater.Data()
		}
		serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
		if err != nil {
			return err
		}
		if err := serv.CheckConfig(finalConfig); err != nil {
			return err
		}
		// 保存配置
		if err := os.WriteFile(singBoxConfigPath, finalConfig, 0660); err != nil {
			return fmt.Errorf("save final config error:\n\t%w", err)
		}
		// 重启服务
		if restoreFlagRestart {
			if err := serv.Restart(); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	restoreCmd.Flags().BoolVarP(&restoreFlagFormat, "format", "f", false, "format config")
	restoreCmd.Flags().BoolVarP(&restoreFlagRestart, "restart", "r", false, "restart service")

	rootCmd.AddCommand(restoreCmd)
}
