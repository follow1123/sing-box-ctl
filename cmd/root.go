package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	Version        = "0.2.0"
	SingBoxVersion = "0.12.x"
)

var rootFlagVersion bool

var rootCmd = &cobra.Command{
	Use:   "sbctl",
	Short: "sing-box helper",
	Run: func(cmd *cobra.Command, args []string) {
		if rootFlagVersion {
			cmd.Printf("version: %s\nsupported sing-box version: %s\n", Version, SingBoxVersion)
			return
		}
		cmd.Help()
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
