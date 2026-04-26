package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/spf13/cobra"
)

var (
	restoreFlagFormat  bool
	restoreFlagRestart bool
)

var restoreCmd = &cobra.Command{
	Use:          "restore",
	Short:        "Restore config from last provider archived config",
	SilenceUsage: true, // 关闭错误时的帮助信息
	GroupID:      cmdGrpDefault,
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Default()
		if err != nil {
			return err
		}
		// 获取最新的归档配置
		archiver, err := archiver.New(conf.ArchiveDir)
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

		conv, err := converter.New(conf.SingBoxTemplateConfigFile)
		if err != nil {
			return err
		}
		sb, err := conv.Convert(data)
		if err != nil {
			return err
		}

		s, err := settings.NewSettings(conf.SingBox.ConfigFile)
		if err != nil {
			return err
		}
		if err := s.UpdateFrom(sb); err != nil {
			return err
		}
		finalData, err := s.ToJson(providerFetchFlagFormat)
		if err != nil {
			return err
		}

		serv := service.New(conf, s)
		if err := serv.CheckConfig(finalData); err != nil {
			return err
		}
		// 保存配置
		if err := s.Save(providerFetchFlagFormat); err != nil {
			return err
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
