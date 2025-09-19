package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var serviceName = "sing-box.service"

type service struct {
	binaryPath string
	configPath string
	workingDir string
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
	// 服务未注册，注册服务
	cmd := exec.Command("systemctl", "status", serviceName)
	// 服务存在返回nil不做任何操作
	if err := cmd.Run(); err != nil {
		var serviceData []byte
		serviceData = fmt.Appendf(serviceData, `[Unit]
Description=sing-box service
Documentation=https://sing-box.sagernet.org
After=network.target nss-lookup.target network-online.target

[Service]
User=sing-box
StateDirectory=sing-box
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
ExecStart=%s -D %s -C %s run
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=10s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target`, binaryPath, workingDir, configPath)

		servicePath := fmt.Sprintf("/etc/systemd/system/%s", serviceName)
		if err := os.WriteFile(servicePath, serviceData, 0660); err != nil {
			return nil, fmt.Errorf("create '%s' service file error:\n\t%w", servicePath, err)
		}
		// 创建服务
		cmd = exec.Command("systemctl", "daemon-reload")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("%w\n%s", err, stderr.String())
		}
	}
	return &service{
		binaryPath: binaryPath,
		configPath: configPath,
		workingDir: workingDir,
	}, nil
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

func (s *service) Restart() error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *service) Start() error {
	cmd := exec.Command("systemctl", "start", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *service) Stop() error {
	cmd := exec.Command("systemctl", "stop", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w\n%s", err, stderr.String())
	}
	return nil
}

func (s *service) IsRunning() bool {
	cmd := exec.Command("systemctl", "status", serviceName)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func downloadCore(binaryPath string) error {
	// 下载压缩包
	url := "https://github.com/SagerNet/sing-box/releases/download/v1.12.8/sing-box-1.12.8-linux-amd64.tar.gz"
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
	log.Printf("decompression sing-box core binary to '%s' ...\n", binaryPath)
	targetFile := "sing-box-1.12.8-linux-amd64/sing-box"
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return fmt.Errorf("read core gz file error:\n\t%w", err)
	}
	gzipReader, err := gzip.NewReader(&buf)
	if err != nil {
		return fmt.Errorf("create gzip reader error:\n\t%w", err)
	}
	tarReader := tar.NewReader(gzipReader)
	var found bool
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // 读完了
		}
		if err != nil {
			return fmt.Errorf("reader tar content error:\n\t%w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue // 跳过非普通文件
		}

		// 找到目标文件，保存内容
		if header.Name == targetFile {
			outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY, 0750)
			if err != nil {
				return fmt.Errorf("create out file error:\n\t%w", err)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("copy target file '%s' error:\n\t%w", targetFile, err)
			}
			found = true
			break
		}
	}
	if !found {
		log.Printf("target file '%s' not found from gz file", targetFile)
	}
	return nil
}
