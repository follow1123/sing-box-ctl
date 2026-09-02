package settings

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	C "github.com/follow1123/sing-box-ctl/converter"
)

type SettingName string

const (
	// sing-box api service（services 中 type=api）
	StAPIStatus    SettingName = "api.status"
	StAPIListen    SettingName = "api.listen"
	StAPIPort      SettingName = "api.port"
	StAPISecret    SettingName = "api.secret"
	StAPIDashboard SettingName = "api.dashboard"
	// mixed inbound
	StMixedStatus         SettingName = "mixed.status"
	StMixedListen         SettingName = "mixed.listen"
	StMixedPort           SettingName = "mixed.port"
	StMixedSysProxyStatus SettingName = "mixed.system.proxy.status"
	// tun inbound
	StTunStatus SettingName = "tun.status"
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

// New 构建设置器。tmplConf 为模板原始配置（提供默认 mixed/tun inbound、api service）；
// conf 为订阅转换后的配置（将被修改）。api service 与 mixed/tun inbound 默认移除
// （禁用），需要时通过 Set 启用。
func New(tmplConf *C.SingBox, conf *C.SingBox) *Settings {
	removeAPIService(conf)
	removeInboundByType(conf, "mixed")
	removeInboundByType(conf, "tun")
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
	case StAPIStatus:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid api status: %s %w\n\t", value, err)
		}
		if status {
			s.initAPIService()
		} else {
			removeAPIService(s.conf)
		}
	case StAPIListen:
		idx := s.initAPIService()
		s.conf.Services[idx]["listen"] = value
	case StAPIPort:
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid api port: %s %w\n\t", value, err)
		}
		idx := s.initAPIService()
		s.conf.Services[idx]["listen_port"] = port
	case StAPISecret:
		idx := s.initAPIService()
		s.conf.Services[idx]["secret"] = value
	case StAPIDashboard:
		status, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid api dashboard status: %s %w\n\t", value, err)
		}
		idx := s.initAPIService()
		dashboard := apiServiceDashboard(s.conf.Services[idx])
		dashboard["enabled"] = status
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
	case StMixedListen:
		idx := s.initMixedSettings()
		s.conf.Inbounds[idx]["listen"] = value
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
	case StAPIStatus:
		return findAPIService(s.conf) >= 0, nil
	case StAPIDashboard:
		idx := findAPIService(s.conf)
		if idx < 0 {
			return false, fmt.Errorf("api service is not enabled")
		}
		dashboard, exists := s.conf.Services[idx]["dashboard"]
		if !exists {
			// 模板未配置 dashboard 时按 sing-box 默认（启用）处理
			return true, nil
		}
		switch v := dashboard.(type) {
		case map[string]any:
			enabled, ok := v["enabled"].(bool)
			if !ok {
				return true, nil
			}
			return enabled, nil
		case bool:
			return v, nil
		default:
			// dashboard 配置为 string（面板目录）时视为启用
			return true, nil
		}
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
	case StAPIPort:
		idx := findAPIService(s.conf)
		if idx < 0 {
			return 0, fmt.Errorf("api service is not enabled")
		}
		return uint16FromAny(s.conf.Services[idx]["listen_port"])
	case StMixedPort:
		idx := indexOfInboundType(s.conf, "mixed")
		if idx < 0 {
			return 0, fmt.Errorf("mixed mode is not enabled")
		}
		portValue, exists := s.conf.Inbounds[idx]["listen_port"]
		if !exists {
			return 0, fmt.Errorf("no listen port in mixed inbound")
		}
		return uint16FromAny(portValue)
	default:
		return 0, fmt.Errorf("invalid name %s", name)
	}
}

func (s *Settings) GetString(name SettingName) (string, error) {
	if s.conf == nil {
		return "", fmt.Errorf("config not init")
	}
	switch name {
	case StAPISecret:
		idx := findAPIService(s.conf)
		if idx < 0 {
			return "", fmt.Errorf("api service is not enabled")
		}
		secret, _ := s.conf.Services[idx]["secret"].(string)
		return secret, nil
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

// initAPIService 确保 conf 中存在 api service；没有时优先复制模板中的，
// 模板也没有时构造默认配置。
func (s *Settings) initAPIService() int {
	if idx := findAPIService(s.conf); idx >= 0 {
		return idx
	}
	if tmplIdx := findAPIService(s.tmplConf); tmplIdx >= 0 {
		s.conf.Services = append(s.conf.Services, cloneService(s.tmplConf.Services[tmplIdx]))
		return len(s.conf.Services) - 1
	}
	s.conf.Services = append(s.conf.Services, defaultAPIService())
	return len(s.conf.Services) - 1
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

// findAPIService 返回 conf 中 type=api 的 service 下标，不存在返回 -1
func findAPIService(sb *C.SingBox) int {
	for i, service := range sb.Services {
		if service["type"] == "api" {
			return i
		}
	}
	return -1
}

// removeAPIService 移除 conf 中所有 type=api 的 service（禁用 api service）
func removeAPIService(sb *C.SingBox) {
	for i := 0; i < len(sb.Services); i++ {
		if sb.Services[i]["type"] == "api" {
			sb.Services = slices.Delete(sb.Services, i, i+1)
			i--
		}
	}
}

// removeInboundByType 移除 conf 中指定类型的所有 inbound（可选模式默认关闭）
func removeInboundByType(sb *C.SingBox, inboundType string) {
	for i := 0; i < len(sb.Inbounds); i++ {
		if sb.Inbounds[i]["type"] == inboundType {
			sb.Inbounds = slices.Delete(sb.Inbounds, i, i+1)
			i--
		}
	}
}

// apiServiceDashboard 获取（必要时创建）api service 的 dashboard 配置对象
func apiServiceDashboard(service map[string]any) map[string]any {
	if dashboard, ok := service["dashboard"].(map[string]any); ok {
		return dashboard
	}
	dashboard := map[string]any{}
	service["dashboard"] = dashboard
	return dashboard
}

// defaultAPIService 模板中没有 api service 时的兜底默认
func defaultAPIService() map[string]any {
	return map[string]any{
		"type":        "api",
		"listen":      "127.0.0.1",
		"listen_port": 9090,
		"dashboard":   map[string]any{"enabled": true},
	}
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

// cloneService 深拷贝 api service 配置，避免修改污染模板配置
func cloneService(src map[string]any) map[string]any {
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return src
	}
	return out
}

// uint16FromAny 将配置中的端口值（json number/整数）转换为 uint16
func uint16FromAny(value any) (uint16, error) {
	portStr := fmt.Sprintf("%v", value)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("parse listen port error:\n\t%w", err)
	}
	return uint16(port), nil
}
