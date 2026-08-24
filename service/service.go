package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/follow1123/sing-box-ctl/config"
)

type SingBoxService struct {
	conf *config.Config
}

func New(conf *config.Config) *SingBoxService {
	return &SingBoxService{conf: conf}
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
