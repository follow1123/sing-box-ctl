package cmd

import (
	"fmt"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	P "github.com/follow1123/sing-box-ctl/provider"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/spf13/cobra"
)

var (
	providerFetchFlagFormat bool
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
		provider, err := P.New(conf.ConfigFile)
		if err != nil {
			return err
		}
		d := provider.GetDefault()
		if d == nil {
			return fmt.Errorf("no default provider")
		}
		url := d.Url

		// 下载远程配置
		data, err := P.DataFromSource(url)
		if err != nil {
			return err
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

		serv := service.New(conf)
		if err := serv.CheckConfig(finalData); err != nil {
			return err
		}
		// 保存配置
		if err := s.Save(providerFetchFlagFormat); err != nil {
			return err
		}

		// 归档下载的原始配置文件
		archiver, err := archiver.New(conf.ArchiveDir)
		if err != nil {
			return err
		}
		if err := archiver.Save(data); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	providerFetchCmd.Flags().BoolVarP(&providerFetchFlagFormat, "format", "f", false, "format config")

	providerCmd.AddCommand(providerFetchCmd)
}
