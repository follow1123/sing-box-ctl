package config

import (
	_ "embed"
	"os/exec"
)

var (
	ConfigHome    = "$HOME/.config/singboxctl"
	ServiceScript = "singbox_service.sh"
	ScriptShell   = []string{"bash"}
)

//go:embed singbox_service.sh
var singboxServiceScriptData []byte

func SetCmdAttr(cmd *exec.Cmd) {
}
