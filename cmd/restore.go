package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/follow1123/sing-box-ctl/archiver"
	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/service"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/goccy/go-yaml"
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
		archiver, err := archiver.New(conf.ArchiveDir())
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

			newConfig.Experimental.ClashAPI = oldConfig.Experimental.ClashAPI
			newConfig.Inbounds = oldConfig.Inbounds
		}

		var finalData []byte
		if restoreFlagFormat {
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
