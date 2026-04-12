package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/provider"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/goccy/go-yaml"
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
		p, err := provider.New(conf.ConfigPath())
		if err != nil {
			return err
		}
		d := p.GetDefault()
		if d == nil {
			return fmt.Errorf("no default provider")
		}
		url := d.Url

		// 下载远程配置
		data, err := provider.DataFromSource(url)
		if err != nil {
			return err
		}

		c := &converter.Clash{}
		if err := yaml.Unmarshal(data, c); err != nil {
			return fmt.Errorf("unmarshal clash yaml config error:\n\t%w", err)
		}

		// 转换成 sing-box 配置
		tmplConf, err := settings.LoadConfigFromPath(conf.SingBoxTmplConfigPath())
		if err != nil {
			return err
		}

		newConfig, err := converter.Convert(c, tmplConf)
		if err != nil {
			return err
		}

		_, err = os.Stat(conf.SingBoxConfigPath())
		var notExists bool
		if err != nil {
			if os.IsNotExist(err) {
				notExists = true
			} else {
				return fmt.Errorf("check sing box config error:\n\t%w", err)
			}
		}

		settings.LoadConfigFromPath(conf.SingBoxConfigPath())

		var finalConfig *converter.SingBox
		if notExists {
			finalConfig = newConfig
		} else {
			oldConfig, err := settings.LoadConfigFromPath(conf.SingBoxConfigPath())
			if err != nil {
				return fmt.Errorf("load old config error:\n\t%w", err)
			}
			fmt.Printf("oldConfig: %v\n", oldConfig)

			newConfig.Experimental.ClashAPI = oldConfig.Experimental.ClashAPI
			newConfig.Inbounds = oldConfig.Inbounds
		}

		var finalData []byte
		if providerFetchFlagFormat {
			finalData, err = json.MarshalIndent(finalConfig, "", "  ")
		} else {
			finalData, err = json.Marshal(finalConfig)
		}
		if err != nil {
			return fmt.Errorf("marshal to json config error:\n\t%w", err)
		}

		serv, err := service.New(conf.SingBoxBinaryPath(), conf.SingBoxConfigPath(), conf.SingBoxWorkingDir())
		if err != nil {
			return err
		}
		if err := serv.CheckConfig(finalData); err != nil {
			return err
		}

		// 保存配置
		if err := os.WriteFile(conf.SingBoxConfigPath(), finalData, 0660); err != nil {
			return fmt.Errorf("save final config error:\n\t%w", err)
		}

		// 归档下载的原始配置文件
		archiver, err := archiver.New(conf.ArchiveDir())
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
