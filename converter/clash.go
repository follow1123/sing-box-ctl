package converter

type ClashConfig struct {
	Rules   []string `yaml:"rules"`
	Proxies []Proxy  `yaml:"proxies"`
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
