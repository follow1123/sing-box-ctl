package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	C "github.com/follow1123/sing-box-ctl/converter"
)

type Platform = string

const (
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformAndroid Platform = "android"
)

type Settings struct {
	tmplConf *C.SingBox
	conf     *C.SingBox
	confPath string
}

func NewSettings(tmplConfigPath string, configPath string) (*Settings, error) {
	tmpConf, err := LoadConfigFromPath(tmplConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load template config error:\n\t%w", err)
	}

	conf, err := LoadConfigFromPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config error:\n\t%w", err)
	}

	return &Settings{tmplConf: tmpConf, conf: conf, confPath: configPath}, nil
}

type MixedProxySettings struct {
	Reset             bool
	Enabled           bool
	Port              *uint16
	AllowLAN          *bool
	EnableSystemProxy *bool
}

type WebUISettings struct {
	Reset    bool
	Enabled  bool
	Addr     *string
	Password *string
}

func (s *Settings) resetWebUISettings() {
	s.conf.Experimental.ClashAPI = s.tmplConf.Experimental.ClashAPI
}

func (s *Settings) UpdateWebUISettings(setting WebUISettings) {
	if setting.Reset {
		s.resetWebUISettings()
		return
	}

	if setting.Enabled {
		if s.conf.Experimental.ClashAPI == nil {
			s.resetWebUISettings()
		}

		if setting.Addr != nil {
			s.conf.Experimental.ClashAPI.ExternalController = *setting.Addr
		}
		if setting.Password != nil {
			s.conf.Experimental.ClashAPI.Secret = *setting.Password
		}
	} else {
		s.conf.Experimental.ClashAPI = nil
	}
}

func (s *Settings) resetMixedProxySettings() {
	defaultInboundIdx := IndexOfInboundType(s.tmplConf, "mixed")
	if defaultInboundIdx < 0 {
		panic("no default mixed inbound config in config")
	}
	inboundIdx := IndexOfInboundType(s.conf, "mixed")
	if inboundIdx >= 0 {
		s.conf.Inbounds[inboundIdx] = s.tmplConf.Inbounds[defaultInboundIdx]
	} else {
		s.conf.Inbounds = append(s.conf.Inbounds, s.tmplConf.Inbounds[defaultInboundIdx])
	}
}

func (s *Settings) UpdateMixedProxySettings(setting MixedProxySettings) error {
	if setting.Reset {
		s.resetMixedProxySettings()
		return nil
	}
	var inboundIdx int = IndexOfInboundType(s.conf, "mixed")
	if inboundIdx < 0 {
		return fmt.Errorf("no mixed inbound found")
	}
	if setting.Enabled {
		if setting.Port != nil {
			s.conf.Inbounds[inboundIdx]["listen_port"] = *setting.Port
		}
		if setting.AllowLAN != nil {
			if *setting.AllowLAN {
				s.conf.Inbounds[inboundIdx]["listen"] = "::"
			} else {
				s.conf.Inbounds[inboundIdx]["listen"] = "127.0.0.1"
			}
		}
		if setting.EnableSystemProxy != nil {
			s.conf.Inbounds[inboundIdx]["set_system_proxy"] = *setting.EnableSystemProxy
		}
	} else {
		s.conf.Inbounds = slices.Delete(s.conf.Inbounds, inboundIdx, inboundIdx+1)
	}
	return nil
}

func (s *Settings) UpdateTunSettings(enable bool) {
	defaultInboundIdx := IndexOfInboundType(s.tmplConf, "tun")
	if defaultInboundIdx < 0 {
		panic("no default tun inbound config in config template")
	}
	var inboundIdx int = IndexOfInboundType(s.conf, "tun")
	if enable {
		if inboundIdx >= 0 {
			return
		}
		s.conf.Inbounds = append(s.conf.Inbounds, s.tmplConf.Inbounds[defaultInboundIdx])
	} else {
		if inboundIdx < 0 {
			return
		}
		s.conf.Inbounds = slices.Delete(s.conf.Inbounds, inboundIdx, inboundIdx+1)
	}
}

func (s *Settings) UpdatePlatformSettings(platform Platform) error {
	var inboundIdx int = IndexOfInboundType(s.conf, "tun")
	switch platform {
	case PlatformWindows:
		if inboundIdx >= 0 {
			s.conf.Inbounds[inboundIdx]["stack"] = "gvisor"
		}
	case PlatformLinux:
		if inboundIdx > 0 {
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

func (s *Settings) Save(format bool) error {
	data, err := s.ToJson(format)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.confPath, data, 0660); err != nil {
		panic(fmt.Errorf("write config data to file %s error:\n\t%w", s.confPath, err))
	}
	return nil
}

func IndexOfInboundType(sb *C.SingBox, inboundType string) int {
	for i, inbound := range sb.Inbounds {
		if inbound["type"] == inboundType {
			return i
		}
	}
	return -1
}

func LoadConfigFromPath(configPath string) (*C.SingBox, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read %s error:\n\t%w", configPath, err)
	}
	sb := &C.SingBox{}
	if err := json.Unmarshal(data, sb); err != nil {
		return nil, fmt.Errorf("unmarshal json file %s error: \n\t%w", configPath, err)
	}
	return sb, nil
}
