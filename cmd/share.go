package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/follow1123/sing-box-ctl/httpshare"
	"github.com/follow1123/sing-box-ctl/platform"
	"github.com/spf13/cobra"
)

const defaultPort uint16 = 45728

var (
	shareFlagOpen bool
	shareFlagPort uint16
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share configuration via http",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := httpshare.New(shareFlagPort)
		if err != nil {
			return err
		}
		if err := s.PrintHelp(); err != nil {
			return err
		}

		go func() {
			log.Printf("start share server on %s ...\n", s.Url())
			if err := s.Share(); err != nil {
				log.Fatal(err)
			}
		}()

		if shareFlagOpen {
			platform.OpenUrl(s.Url())
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Stop(ctx); err != nil {
			log.Fatal("Server forced to shutdown: ", err)
		}

		log.Println("Server exiting")
		return nil
	},
}

func init() {
	shareCmd.Flags().BoolVarP(&shareFlagOpen, "open", "o", false, "open with default browser")
	shareCmd.Flags().Uint16VarP(&shareFlagPort, "port", "p", defaultPort, "port")
	rootCmd.AddCommand(shareCmd)
}
