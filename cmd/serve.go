package cmd

import (
	"github.com/follow1123/sing-box-ctl/webui"
)

func serveCmd(workingDir, host string, port int) error {
	server, err := webui.New(workingDir, host, port)
	if err != nil {
		return err
	}
	return server.Serve()
}
