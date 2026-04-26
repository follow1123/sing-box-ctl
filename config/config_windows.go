package config

import (
	_ "embed"
	"os/exec"
	"runtime"
	"syscall"
)

var (
	ConfigHome    = "$LOCALAPPDATA/singboxctl"
	ServiceScript = "singbox_servcie.ps1"
	ScriptShell   = []string{"powershell.exe", "-NoProfile", "-ExecutionPolicy", "RemoteSigned", "-File"}
)

//go:embed singbox_service.ps1
var singboxServiceScriptData []byte

func SetCmdAttr(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}
