package cmd

import (
	"context"
	"os/signal"
	"strings"
	"syscall"

	"github.com/follow1123/sing-box-ctl/clash"
	"github.com/follow1123/sing-box-ctl/config"
	L "github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/services"
	"github.com/follow1123/sing-box-ctl/singbox"
	"github.com/spf13/cobra"
)

var (
	serveFlagUpdater   string
	serveFlagGenerator string
	serveFlagWrite     bool
	serveFlagLogFlie   string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start http server generate config and a ticker update remote config",
	Run: func(cmd *cobra.Command, args []string) {
		log := L.NewCliLogger()

		cm := config.CreateConfManager()
		conf, err := cm.Load()
		if err != nil {
			log.Error(err)
			return
		}

		changeOpts := make(map[string]any)

		if serveFlagUpdater != "" {
			switch serveFlagUpdater {
			case "true":
				changeOpts["$.services.updater.enabled"] = true
				conf.Services.Updater.Enabled = true
			case "false":
				changeOpts["$.services.updater.enabled"] = false
				conf.Services.Updater.Enabled = false
			default:
				changeOpts["$.services.updater.enabled"] = true
				conf.Services.Updater.Enabled = true

				updaterConfs := strings.Split(serveFlagUpdater, "+")
				for i, v := range updaterConfs {
					if i == 0 {
						changeOpts["$.services.updater.url"] = v
						conf.Services.Updater.Url = v
					}
					if i == 1 {
						changeOpts["$.services.updater.interval"] = v
						conf.Services.Updater.Interval = v
					}
				}
			}
		}

		if serveFlagGenerator != "" {
			switch serveFlagGenerator {
			case "true":
				changeOpts["$.services.generator.enabled"] = true
				conf.Services.Generator.Enabled = true
			case "false":
				changeOpts["$.services.generator.enabled"] = false
				conf.Services.Generator.Enabled = false
			default:
				changeOpts["$.services.generator.enabled"] = true
				changeOpts["$.services.generator.addr"] = serveFlagGenerator
				conf.Services.Generator.Enabled = true
				conf.Services.Generator.Addr = serveFlagGenerator
			}
		}

		// 只写入配置，不启动服务
		if serveFlagWrite {
			if err := cm.ChangeConfig(changeOpts); err != nil {
				log.Error(err)
			}
			return
		}

		sl := L.NewServiceLogger(serveFlagLogFlie)
		defer sl.Sync()

		clashUpdater, err := clash.NewClashUpdater(
			sl,
			cm.GetConfigHome(),
			conf.Services.Updater.Url,
			conf.Services.Updater.Interval,
		)
		if err != nil {
			log.Error(err)
			return
		}

		renderer, err := singbox.NewConfRenderer(sl, cm.GetTemplateFile())
		if err != nil {
			log.Error(err)
			return
		}

		sb := singbox.NewSingBox(sl, renderer, conf.SingBox)
		generator := services.NewGenerator(sl, sb, clashUpdater, conf.Services.Generator)
		updater, err := services.NewUpdater(sl, sb, clashUpdater)
		if err != nil {
			log.Error(err)
			return
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		if conf.Services.Updater.Enabled {
			go updater.Start()
			defer updater.Stop()
		}
		if conf.Services.Generator.Enabled {
			go func() {
				if err := generator.Start(); err != nil {
					log.Error(err)
					cancel()
					return
				}
			}()
			defer generator.Stop()
		}
		<-ctx.Done()
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveFlagUpdater, "updater", "u", "", "set updater configuration, available value: [true|false|<url+interval>]")
	serveCmd.Flags().StringVarP(&serveFlagGenerator, "generator", "g", "", "set generator configuration, available value: [true|false|<address>]")
	serveCmd.Flags().BoolVarP(&serveFlagWrite, "write", "W", false, "write to local configuration")
	serveCmd.Flags().StringVarP(&serveFlagLogFlie, "log-file", "l", "", "set log file full path")
	rootCmd.AddCommand(serveCmd)
}
