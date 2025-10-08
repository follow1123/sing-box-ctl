package cmd

import (
	"fmt"
	"os"

	A "github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/follow1123/sing-box-ctl/service"
	U "github.com/follow1123/sing-box-ctl/updater"
	"github.com/spf13/cobra"
)

var (
	providerFetchFlagFormat  bool
	providerFetchFlagRestart bool
)

var providerFetchCmd = &cobra.Command{
	Use:          "fetch",
	Short:        "Fetch and convert config from default provider",
	SilenceUsage: true, // 关闭错误时的帮助信息
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Default()
		if err != nil {
			return err
		}
		provider, err := P.New(conf.ConfigPath())
		if err != nil {
			return err
		}
		d, err := provider.GetDefault()
		if err != nil {
			return err
		}
		url := d.Url

		// 下载远程配置
		data, err := P.DataFromSource(url)
		if err != nil {
			return err
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
			if err := updater.Upgrade(newConfig, providerFetchFlagFormat); err != nil {
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

		// 归档下载的原始配置文件
		archiver, err := A.New(conf.ArchiveDir())
		if err != nil {
			return err
		}
		if err := archiver.Save(data); err != nil {
			return err
		}
		// 重启服务
		if providerFetchFlagRestart {
			if err := serv.Restart(); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	providerFetchCmd.Flags().BoolVarP(&providerFetchFlagFormat, "format", "f", false, "format config")
	providerFetchCmd.Flags().BoolVarP(&providerFetchFlagRestart, "restart", "r", false, "restart service")

	providerCmd.AddCommand(providerFetchCmd)
}
