package service

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/follow1123/sing-box-ctl/jsonhandler"
)

type service struct {
	binaryPath    string
	configPath    string
	workingDir    string
	powershellCmd []string
}

func New(binaryPath string, configPath string, workingDir string) (Service, error) {
	// 下载 sing-box 内核
	_, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := downloadCore(binaryPath); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("check binary path '%s' error:\n\t%w", binaryPath, err)
		}
	}
	return &service{
		binaryPath:    binaryPath,
		configPath:    configPath,
		workingDir:    workingDir,
		powershellCmd: []string{"powershell.exe", "-NoProfile", "-ExecutionPolicy", "RemoteSigned", "-Command"},
	}, nil
}

func (s *service) Start() error {
	// 正在运行，直接退出
	if s.IsRunning() {
		return nil
	}
	isTunMode, err := s.isTunMode()
	if err != nil {
		return fmt.Errorf("check is tun mode error:\n\t%w", err)
	}
	if isTunMode {
		// tun 模式启动先关闭系统代理
		if err := s.turnOffSystemProxy(); err != nil {
			return fmt.Errorf("turn off system proxy error:\n\t%w", err)
		}
		return s.start(true)
	} else {
		return s.start(false)
	}
}

func (s *service) Stop() error {
	pid, err := s.pid()
	// 查不到 pid 直接退出
	if err != nil {
		return nil
	}
	isTunMode, err := s.isTunMode()
	if err != nil {
		return fmt.Errorf("check is tun mode error:\n\t%w", err)
	}
	if err := s.stop(pid, isTunMode, false); err != nil {
		return err
	}
	// 关闭系统代理
	if err := s.turnOffSystemProxy(); err != nil {
		return fmt.Errorf("turn off system proxy error:\n\t%w", err)
	}
	return nil
}

func (s *service) Restart() error {
	isTunMode, err := s.isTunMode()
	if err != nil {
		return fmt.Errorf("check is tun mode error:\n\t%w", err)
	}
	pid, err := s.pid()
	// 查到 pid 才执行结束命令
	if err == nil {
		if err := s.stop(pid, isTunMode, false); err != nil {
			return err
		}
		// 关闭系统代理
		if err := s.turnOffSystemProxy(); err != nil {
			return fmt.Errorf("turn off system proxy error:\n\t%w", err)
		}
		// 睡眠 500 毫秒
		time.Sleep(500 * time.Millisecond)
	}
	return s.start(isTunMode)
}

func (s *service) IsRunning() bool {
	_, err := s.pid()
	return err == nil
}

func (s *service) CheckConfig(data []byte) error {
	f, err := os.CreateTemp("", "sing-box-config-*.json")
	if err != nil {
		return fmt.Errorf("create temp config to check error:\n\t%w", err)
	}
	defer os.Remove(f.Name()) // 程序退出时自动删除
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write temp config data error:\n\t%w", err)
	}
	cmd := exec.Command(s.binaryPath, "check", "-c", f.Name())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("check config error\n\t%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *service) pid() (string, error) {
	command := append([]string{}, s.powershellCmd...)
	command = append(command, "(Get-Process -Name sing-box)[0].Id")
	cmd := exec.Command(command[0], command[1:]...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (s *service) start(admin bool) error {
	command := append([]string{}, s.powershellCmd...)
	var adminFlag = ""
	if admin {
		adminFlag = " -Verb RunAs"
	}
	command = append(command, fmt.Sprintf(
		`Start-Process -FilePath %s -ArgumentList "-D %s", "-c %s", "run" -WindowStyle Hidden%s`,
		s.binaryPath,
		s.workingDir,
		s.configPath,
		adminFlag,
	))
	cmd := exec.Command(command[0], command[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}

func (s *service) stop(pid string, admin bool, force bool) error {
	command := []string{"taskkill", "/pid", pid}
	if force {
		command = append(command, "/f")
	}
	if admin {
		command = append([]string{}, s.powershellCmd...)
		if force {
			command = append(command, fmt.Sprintf(
				`Start-Process -FilePath taskkill -ArgumentList "/pid %s", "/f" -WindowStyle Hidden -Verb RunAs`,
				pid,
			))
		} else {
			command = append(command, fmt.Sprintf(
				`Start-Process -FilePath taskkill -ArgumentList "/pid %s" -WindowStyle Hidden -Verb RunAs`,
				pid,
			))
		}
	}
	cmd := exec.Command(command[0], command[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}

func (s *service) isTunMode() (bool, error) {
	jh, err := jsonhandler.FromFile(s.configPath)
	if err != nil {
		return false, err
	}
	inboundType, exists := jh.GetString("inbounds.0.type")
	if !exists {
		return false, errors.New("no inbound or no inbound type")
	}
	return inboundType == "tun", nil
}

func (s *service) turnOffSystemProxy() error {
	command := append([]string{}, s.powershellCmd...)
	command = append(command, `(Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyEnable).ProxyEnable`)
	cmd := exec.Command(command[0], command[1:]...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	isProxyEnabled := strings.TrimSpace(stdout.String()) == "0"
	if isProxyEnabled {
		return nil
	}
	command[len(command)-1] = `Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyEnable -Value 0`
	cmd = exec.Command(command[0], command[1:]...)
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run command '%s' error:\n\t%w", cmd.String(), err)
	}
	return nil
}

func downloadCore(binaryPath string) error {
	// 下载压缩包
	url := "https://github.com/SagerNet/sing-box/releases/download/v1.12.8/sing-box-1.12.8-windows-amd64.zip"
	log.Printf("Download sing-box core from '%s' ...\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download core from '%s' error:\n\t%w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download core error code %d", resp.StatusCode)
	}
	// 解压
	log.Printf("unzip sing-box core binary to '%s' ...\n", binaryPath)
	targetFile := "sing-box-1.12.8-windows-amd64/sing-box.exe"
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read core zip error:\n\t%w", err)
	}
	r := bytes.NewReader(data)
	zipReader, err := zip.NewReader(r, int64(len(data)))
	if err != nil {
		return fmt.Errorf("create zip reader error:\n\t%w", err)
	}
	var found bool
	for _, file := range zipReader.File {
		if file.Name == targetFile {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("open target file '%s' error:\n\t%w", targetFile, err)
			}
			defer rc.Close()
			outFile, err := os.Create(binaryPath)
			if err != nil {
				return fmt.Errorf("create out file error:\n\t%w", err)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, rc); err != nil {
				return fmt.Errorf("copy target file '%s' error:\n\t%w", targetFile, err)
			}
			found = true
			break
		}
	}
	if !found {
		log.Printf("target file '%s' not found from zip", targetFile)
	}
	return nil
}
