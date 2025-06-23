package clash

import (
	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
	"github.com/goccy/go-yaml"
)

type ClashConfig struct {
	Rules       []string     `yaml:"rules"`
	Proxies     []Proxy      `yaml:"proxies"`
	ProxyGroups []ProxyGroup `yaml:"proxy-groups"`
}

func NewClashConfig(data []byte) *ClashConfig {
	cc := &ClashConfig{}
	if err := yaml.Unmarshal(data, &cc); err != nil {
		// 忽略其他属性
	}
	return cc
}

type Proxy struct {
	Type string `yaml:"type"`
	// shadowsocks、trojan 协议通用属性
	Name     string `yaml:"name"`
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`

	// shadowsocks 协议属性
	Cipher string `yaml:"cipher"`
	Udp    bool   `yaml:"udp"`

	// trojan 协议属性
	Sni            string `yaml:"sni"`
	SkipCertVerify bool   `yaml:"skip-cert-verify"`
}

type ProxyGroup struct {
	Type string `yaml:"type"`

	// select、url-test 通用属性
	Name    string   `yaml:"name"`
	Proxies []string `yaml:"proxies"`

	// url-test 属性
	Url       string `yaml:"url"`
	Interval  int    `yaml:"interval"`
	Tolerance int    `yaml:"tolerance"`
}

func (cc *ClashConfig) Convert(log logger.Logger) *singbox.PartialConfig {
	pc := singbox.NewPartialConfig()
	convertProxies(log, cc, pc)
	convertProxyGroups(log, cc, pc)
	convertRules(log, cc, pc)

	return pc
}
