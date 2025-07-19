package cmd

import (
	"os"

	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/spf13/cobra"
)

const (
	Version        = "0.1.0"
	SingBoxVersion = "0.11.x"
)

var rootFlagVersion bool

var rootCmd = &cobra.Command{
	Use:   "sbctl",
	Short: "A sing-box tool, support simple subscription conversion",
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewCliLogger()

		if rootFlagVersion {
			log.Info("version: %s\nsupported sing-box version: %s\n", Version, SingBoxVersion)
			return
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&rootFlagVersion, "version", "v", false, "print version")
}
