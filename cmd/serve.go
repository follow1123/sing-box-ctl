package cmd

import (
	"github.com/follow1123/sing-box-ctl/webui"
)

func serveCmd(configPath string, port int) error {
	server, err := webui.New(configPath, port)
	if err != nil {
		return err
	}
	return server.Serve()
}
