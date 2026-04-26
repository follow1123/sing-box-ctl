package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/settings"
)

type SingBoxService struct {
	conf *config.Config
	sts  *settings.Settings
}

func New(conf *config.Config, sts *settings.Settings) *SingBoxService {
	return &SingBoxService{conf: conf, sts: sts}
}

func (s *SingBoxService) Start() error {
	cmd := s.buildCmd()
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SINGBOX_SERVICE_BINARY=%s", s.conf.SingBox.Binary),
		fmt.Sprintf("SINGBOX_SERVICE_WORKING_DIR=%s", s.conf.SingBox.WorkingDir),
		fmt.Sprintf("SINGBOX_SERVICE_CONFIG_PATH=%s", s.conf.SingBox.ConfigFile),
		"SINGBOX_SERVICE_ACTION=start",
	)
	highPerm, err := s.sts.HaveHighPermSetting()
	if err != nil {
		return fmt.Errorf("check have high permission setting error:\n\t%w", err)
	}
	if highPerm {
		cmd.Env = append(os.Environ(), "SINGBOX_SERVICE_HIGH_PERMISSION=1")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start singbox error\n\t%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *SingBoxService) Stop() error {
	cmd := s.buildCmd()
	cmd.Env = append(
		os.Environ(),
		"SINGBOX_SERVICE_ACTION=stop",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stop singbox error\n\t%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *SingBoxService) Restart() error {
	cmd := s.buildCmd()
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SINGBOX_SERVICE_BINARY=%s", s.conf.SingBox.Binary),
		fmt.Sprintf("SINGBOX_SERVICE_WORKING_DIR=%s", s.conf.SingBox.WorkingDir),
		fmt.Sprintf("SINGBOX_SERVICE_CONFIG_PATH=%s", s.conf.SingBox.ConfigFile),
		"SINGBOX_SERVICE_ACTION=restart",
	)
	highPerm, err := s.sts.HaveHighPermSetting()
	if err != nil {
		return fmt.Errorf("check have high permission setting error:\n\t%w", err)
	}
	if highPerm {
		cmd.Env = append(os.Environ(), "SINGBOX_SERVICE_HIGH_PERMISSION=1")
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restart singbox error\n\t%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *SingBoxService) IsRunning() bool {
	cmd := s.buildCmd()
	cmd.Env = append(
		os.Environ(),
		"SINGBOX_SERVICE_ACTION=is_running",
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(stdout.String()) == "true"
}

func (s *SingBoxService) CheckConfig(data []byte) error {
	f, err := os.CreateTemp("", "sing-box-config-*.json")
	if err != nil {
		return fmt.Errorf("create temp config to check error:\n\t%w", err)
	}
	defer os.Remove(f.Name()) // 程序退出时自动删除
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write temp config data error:\n\t%w", err)
	}
	cmd := exec.Command(s.conf.SingBox.Binary, "check", "-c", f.Name())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("check config error\n\t%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *SingBoxService) buildCmd() *exec.Cmd {
	cmdArgs := append(config.ScriptShell, s.conf.SingBoxServiceScript)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	config.SetCmdAttr(cmd)
	return cmd
}
