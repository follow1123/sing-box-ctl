package singbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/logger"
)

type SubscriptionConfig interface {
	Convert(log logger.Logger) *PartialConfig
}

type SingBox struct {
	log          logger.Logger
	confFilePath string
	checkCmd     string
	restartCmd   string
	options      config.SingBoxOptions
	renderer     *ConfRenderer
}

func NewSingBox(log logger.Logger, renderer *ConfRenderer, conf config.SingBoxConfig) *SingBox {
	return &SingBox{
		log:          log,
		confFilePath: conf.ConfigFile,
		checkCmd:     conf.CheckCommand,
		restartCmd:   conf.RestartCommand,
		options:      conf.Options,
		renderer:     renderer,
	}
}

func (sb *SingBox) Check(configFilePath string) error {
	sb.log.Debug("run check command, file: %v", configFilePath)
	cmdStr := fmt.Sprintf(sb.checkCmd, configFilePath)
	cmdArr := strings.Fields(cmdStr)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdArr[0], cmdArr[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run check command error: \n\t%v\n\n%s", err, stderr.String())
	}
	return nil
}

func (sb *SingBox) Restart() error {
	sb.log.Debug("run restart command")
	cmdArr := strings.Fields(sb.restartCmd)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdArr[0], cmdArr[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run restart command error: \n\t%w\n\n%s", err, stderr.String())
	}
	return nil
}

func (sb *SingBox) checkData(data []byte) error {
	f, err := os.CreateTemp("", "generated_config_")
	if err != nil {
		return fmt.Errorf("create template file error: \n\t%w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			sb.log.Fatal(fmt.Errorf("close template file %q error: \n\t%w", f.Name(), err))
		}

		if err := os.Remove(f.Name()); err != nil {
			sb.log.Fatal(fmt.Errorf("remove template file %q error: \n\t%w", f.Name(), err))
		}
	}()

	sb.log.Debug("create template config to check: %q", f.Name())
	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("write template file %q error: \n\t%w", f.Name(), err)
	}

	return sb.Check(f.Name())
}

func (sb *SingBox) SaveConfig(data []byte) error {
	if err := os.WriteFile(sb.confFilePath, data, 0660); err != nil {
		return fmt.Errorf("write config to %q error: \n\t%w", sb.confFilePath, err)
	}
	sb.log.Debug("config save at %q", sb.confFilePath)
	return nil
}

func (sb *SingBox) ConvertConfig(sc SubscriptionConfig, opts ...Opt) ([]byte, error) {
	proxyMode, err := StrToProxyMode(sb.options.Mode)
	if err != nil {
		return nil, fmt.Errorf("invalid default mode: \n\t%w", err)
	}

	options := &Options{
		Android:      false,
		WebUIEnabled: sb.options.WebUIEnabled,
		WebUIAddr:    sb.options.WebUIAddr,
		Mode:         proxyMode,
	}
	for _, o := range opts {
		o(options)
	}
	data, err := sb.renderer.Render(sc.Convert(sb.log), options)
	if err != nil {
		return nil, err
	}

	if err := sb.checkData(data); err != nil {
		return nil, err
	}
	return data, nil
}
