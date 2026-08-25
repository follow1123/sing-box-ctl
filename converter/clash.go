package converter

// Clash 订阅配置结构，只解析节点（proxies），不解析规则（rules）
type Clash struct {
	Proxies []Proxy `yaml:"proxies"`
}

type Proxy struct {
	Type              string   `yaml:"type"`
	Name              string   `yaml:"name"`
	Server            string   `yaml:"server"`
	Port              int      `yaml:"port"`
	Password          string   `yaml:"password"`
	Cipher            string   `yaml:"cipher"`
	Udp               bool     `yaml:"udp"`
	Sni               string   `yaml:"sni"`
	SkipCertVerify    bool     `yaml:"skip-cert-verify"`
	ClientFingerprint string   `yaml:"client-fingerprint"`
	Alpn              []string `yaml:"alpn"`
}
