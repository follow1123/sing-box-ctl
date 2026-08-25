package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strconv"

	C "github.com/follow1123/sing-box-ctl/converter"
)

type SettingName string

const (
	StWebuiStatus         SettingName = "webui.status"
	StWebuiPort           SettingName = "webui.port"
	StWebuiSecret         SettingName = "webui.secret"
	StMixedStatus         SettingName = "mixed.status"
	StMixedPort           SettingName = "mixed.port"
	StMixedSysProxyStatus SettingName = "mixed.system.proxy.status"
	StMixedShareStatus    SettingName = "mixed.share.status"
	StTunStatus           SettingName = "tun.status"
)

type Platform = string

const (
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformAndroid Platform = "android"
)

// Settings 基于模板配置 + 转换后的配置，在内存中应用 URL 参数
type Settings struct {
	tmplConf *C.SingBox
	conf     *C.SingBox
}

// New 构建设置器。tmplConf 为模板原始配置（提供默认 mixed/tun inbound、clash_api）；
// conf 为订阅转换后的配置（将被修改）。webui（clash_api）默认禁用，需要时通过 Set 启用。
func New(tmplConf *C.SingBox, conf *C.SingBox) *Settings {
	conf.Experimental.ClashAPI = nil
	return &Settings{tmplConf: tmplConf, conf: conf}
}

func (s *Settings) SetMap(values map[SettingName]string) error {
	for k, v := range values {
		if err := s.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Settings) Set(name SettingName, value string) error {
	if s.conf == nil {
		return fmt.Errorf("config not init")
	}
	switch name {
	case StWebuiStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid webui status: %s %w\n\t", value, err)
		}
		if status {
			s.initWebuiSettings()
		} else {
			s.conf.Experimental.ClashAPI = nil
		}
	case StWebuiPort:
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid webui port: %s %w\n\t", value, err)
		}
		s.initWebuiSettings()
		s.conf.Experimental.ClashAPI.ExternalController = fmt.Sprintf("127.0.0.1:%d", port)
	case StWebuiSecret:
		s.initWebuiSettings()
		s.conf.Experimental.ClashAPI.Secret = value
	case StMixedStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid mixed status: %s %w\n\t", value, err)
		}
		idx := indexOfInboundType(s.conf, "mixed")
		if status {
			if idx < 0 {
				defaultIdx := indexOfInboundType(s.tmplConf, "mixed")
				if defaultIdx < 0 {
					return fmt.Errorf("no default mixed inbound config in template config")
				}
				s.conf.Inbounds = append(s.conf.Inbounds, cloneInbound(s.tmplConf.Inbounds[defaultIdx]))
			}
		} else {
			if idx >= 0 {
				s.conf.Inbounds = slices.Delete(s.conf.Inbounds, idx, idx+1)
			}
		}
	case StMixedPort:
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid mixed mode port: %s %w\n\t", value, err)
		}
		idx := s.initMixedSettings()
		s.conf.Inbounds[idx]["listen_port"] = port
	case StMixedSysProxyStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid system proxy status: %s %w\n\t", value, err)
		}
		idx := s.initMixedSettings()
		s.conf.Inbounds[idx]["set_system_proxy"] = status
	case StMixedShareStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid proxy share status: %s %w\n\t", value, err)
		}
		idx := s.initMixedSettings()

		if status {
			s.conf.Inbounds[idx]["listen"] = "::"
		} else {
			s.conf.Inbounds[idx]["listen"] = "127.0.0.1"
		}
	case StTunStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid tun mode status: %s %w\n\t", value, err)
		}

		idx := indexOfInboundType(s.conf, "tun")
		if status {
			if idx < 0 {
				defaultIdx := indexOfInboundType(s.tmplConf, "tun")
				if defaultIdx < 0 {
					return fmt.Errorf("no default tun inbound config in template config")
				}
				s.conf.Inbounds = append(s.conf.Inbounds, cloneInbound(s.tmplConf.Inbounds[defaultIdx]))
			}
		} else {
			if idx >= 0 {
				s.conf.Inbounds = slices.Delete(s.conf.Inbounds, idx, idx+1)
			}
		}
	default:
		return fmt.Errorf("invalid name %s", name)
	}
	return nil
}

func (s *Settings) GetBool(name SettingName) (bool, error) {
	if s.conf == nil {
		return false, fmt.Errorf("config not init")
	}
	switch name {
	case StWebuiStatus:
		return s.conf.Experimental.ClashAPI != nil && s.conf.Experimental.ClashAPI.ExternalController != "", nil
	case StMixedStatus:
		return indexOfInboundType(s.conf, "mixed") >= 0, nil
	case StMixedSysProxyStatus:
		idx := indexOfInboundType(s.conf, "mixed")
		if idx < 0 {
			return false, fmt.Errorf("mixed mode is not enabled")
		}
		systemProxyStatusValue, exists := s.conf.Inbounds[idx]["set_system_proxy"]
		if !exists {
			return false, nil
		}
		systemProxyStatus, ok := systemProxyStatusValue.(bool)
		if !ok {
			return false, fmt.Errorf("invalid system proxy value in config")
		}
		return systemProxyStatus, nil
	case StMixedShareStatus:
		idx := indexOfInboundType(s.conf, "mixed")
		if idx < 0 {
			return false, fmt.Errorf("mixed mode is not enabled")
		}
		listenValue, exists := s.conf.Inbounds[idx]["listen"]
		if !exists {
			return false, nil
		}
		listen, ok := listenValue.(string)
		if !ok {
			return false, fmt.Errorf("invalid listen value in config")
		}
		return listen == "::", nil
	case StTunStatus:
		return indexOfInboundType(s.conf, "tun") >= 0, nil
	default:
		return false, fmt.Errorf("invalid name %s", name)
	}
}

func (s *Settings) GetUint16(name SettingName) (uint16, error) {
	if s.conf == nil {
		return 0, fmt.Errorf("config not init")
	}
	switch name {
	case StWebuiPort:
		if s.conf.Experimental.ClashAPI == nil {
			return 0, fmt.Errorf("webui is not enabled")
		}
		addr := s.conf.Experimental.ClashAPI.ExternalController
		_, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return 0, fmt.Errorf("split host port error:\n\t%w", err)
		}
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("parse port error:\n\t%w", err)
		}
		return uint16(port), nil
	case StMixedPort:
		idx := indexOfInboundType(s.conf, "mixed")
		if idx < 0 {
			return 0, fmt.Errorf("mixed mode is not enabled")
		}
		portValue, exists := s.conf.Inbounds[idx]["listen_port"]
		if !exists {
			return 0, fmt.Errorf("no listen port in mixed inbound")
		}
		portStr := fmt.Sprintf("%v", portValue)

		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("parse mixed inoubnd listen port error:\n\t%w", err)
		}
		return uint16(port), nil
	default:
		return 0, fmt.Errorf("invalid name %s", name)
	}
}

func (s *Settings) GetString(name SettingName) (string, error) {
	if s.conf == nil {
		return "", fmt.Errorf("config not init")
	}
	switch name {
	case StWebuiSecret:
		if s.conf.Experimental.ClashAPI == nil {
			return "", fmt.Errorf("webui is not enabled")
		}
		return s.conf.Experimental.ClashAPI.Secret, nil
	default:
		return "", fmt.Errorf("invalid name %s", name)
	}
}

func (s *Settings) initMixedSettings() int {
	defaultIdx := indexOfInboundType(s.tmplConf, "mixed")
	if defaultIdx < 0 {
		panic("no default mixed inbound config in template config")
	}

	idx := indexOfInboundType(s.conf, "mixed")
	if idx < 0 {
		s.conf.Inbounds = append(s.conf.Inbounds, cloneInbound(s.tmplConf.Inbounds[defaultIdx]))
		return len(s.conf.Inbounds) - 1
	}
	return idx
}

func (s *Settings) initWebuiSettings() {
	// clash_api 为空或 external_controller 为空表示禁用状态
	// 优先复制模板中的配置，模板没有时构造空结构
	if s.conf.Experimental.ClashAPI == nil || s.conf.Experimental.ClashAPI.ExternalController == "" {
		if s.tmplConf.Experimental.ClashAPI != nil {
			s.conf.Experimental.ClashAPI = cloneClashAPI(s.tmplConf.Experimental.ClashAPI)
		} else {
			s.conf.Experimental.ClashAPI = &C.ClashAPI{}
		}
	}
}

func (s *Settings) SetPlatform(platform Platform) error {
	var inboundIdx int = indexOfInboundType(s.conf, "tun")
	switch platform {
	case PlatformWindows:
		if inboundIdx >= 0 {
			s.conf.Inbounds[inboundIdx]["stack"] = "gvisor"
		}
	case PlatformLinux:
		if inboundIdx >= 0 {
			s.conf.Inbounds[inboundIdx]["auto_route"] = true
			s.conf.Inbounds[inboundIdx]["auto_redirect"] = true
		}
	case PlatformAndroid:
		if inboundIdx < 0 {
			return fmt.Errorf("android platform must use tun mode")
		}
		s.conf.Inbounds[inboundIdx]["stack"] = "system"
		s.conf.Route.OverrideAndroidVpn = true
	default:
		return fmt.Errorf("invalid platform: %s", platform)

	}
	return nil

}

func (s *Settings) ToJson(format bool) ([]byte, error) {
	var data []byte
	var err error
	if format {
		data, err = json.MarshalIndent(s.conf, "", "  ")
	} else {
		data, err = json.Marshal(s.conf)
	}
	if err != nil {
		return nil, fmt.Errorf("marshal config to json error:\n\t%w", err)
	}

	return data, nil
}

func (s *Settings) GetConfig() *C.SingBox {
	return s.conf
}

func indexOfInboundType(sb *C.SingBox, inboundType string) int {
	for i, inbound := range sb.Inbounds {
		if inbound["type"] == inboundType {
			return i
		}
	}
	return -1
}

// cloneInbound 深拷贝 inbound，避免修改污染模板配置
func cloneInbound(inb map[string]any) map[string]any {
	data, err := json.Marshal(inb)
	if err != nil {
		return inb
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return inb
	}
	return out
}

// cloneClashAPI 深拷贝 clash_api 配置
func cloneClashAPI(src *C.ClashAPI) *C.ClashAPI {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return &C.ClashAPI{}
	}
	var out C.ClashAPI
	if err := json.Unmarshal(data, &out); err != nil {
		return &C.ClashAPI{}
	}
	return &out
}
