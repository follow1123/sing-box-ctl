package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

type Converter struct {
	tmpl   *SingBox
	custom *Custom
}

func New(singboxTemplateConfigPath string) (*Converter, error) {
	tmplConf, err := LoadSingboxFromPath(singboxTemplateConfigPath)
	if err != nil {
		return nil, err
	}
	custom := tmplConf.Custom
	// 清空自定义配置，防止生成到最终配置内
	tmplConf.Custom = nil
	return &Converter{tmpl: tmplConf, custom: custom}, nil
}

func (c *Converter) Convert(clashData []byte) (*SingBox, error) {
	clash := &Clash{}
	if err := yaml.Unmarshal(clashData, clash); err != nil {
		return nil, fmt.Errorf("unmarshal clash yaml config error:\n\t%w", err)
	}

	if err := c.convertOutbounds(clash); err != nil {
		return nil, fmt.Errorf("convert clash proxy to sing-box outbound error:\n\t%w", err)
	}
	if err := c.convertRules(clash); err != nil {
		return nil, fmt.Errorf("convert clash rule to sing-box rule error:\n\t%w", err)
	}
	c.tmpl.Inbounds = []map[string]any{c.tmpl.Inbounds[c.custom.DefaultInboundIndex]}
	return c.tmpl, nil
}

func (c *Converter) convertRules(clash *Clash) error {
	c.tmpl.DNS.Rules = slices.Insert(c.tmpl.DNS.Rules, c.custom.DirectRuleSetIndexInDNS, map[string]any{
		"rule_set": "providers_default_direct_rules", "server": c.custom.DirectDNSServer,
	})
	c.tmpl.DNS.Rules = slices.Insert(c.tmpl.DNS.Rules, c.custom.ProxyRuleSetIndexInDNS, map[string]any{
		"rule_set": "providers_default_proxy_rules", "server": c.custom.ProxyDNSServer,
	})

	routeDirectRuleSet := make(map[string]any, 0)
	routeProxyRuleSet := make(map[string]any, 0)

	c.tmpl.Route.Rules = slices.Insert(c.tmpl.Route.Rules, c.custom.DirectRuleSetIndexInRoute, map[string]any{
		"rule_set": "providers_default_direct_rules", "outbound": "直连",
	})
	c.tmpl.Route.Rules = slices.Insert(c.tmpl.Route.Rules, c.custom.ProxyRuleSetIndexInRoute, map[string]any{
		"rule_set": "providers_default_proxy_rules", "outbound": "节点选择",
	})
	c.tmpl.Route.RuleSet = append(
		c.tmpl.Route.RuleSet,
		map[string]any{"type": "inline", "tag": "providers_default_direct_rules", "rules": [1]map[string]any{routeDirectRuleSet}},
		map[string]any{"type": "inline", "tag": "providers_default_proxy_rules", "rules": [1]map[string]any{routeProxyRuleSet}},
	)

	for _, r := range clash.Rules {
		items := strings.Split(r, ",")

		if len(items) < 3 {
			fmt.Printf("ignore rule：%v\n", r)
			continue
		}

		name, err := ruleType(items[0])
		if err != nil {
			fmt.Print(err)
			continue
		}

		value := items[1]
		outbound := items[2]
		if isDirect(outbound, c.custom.DirectRuleKeywords) {
			valueList, exists := routeDirectRuleSet[name]
			if exists {
				if valueList, ok := valueList.([]any); ok {
					routeDirectRuleSet[name] = append(valueList, value)
				}
			} else {
				routeDirectRuleSet[name] = []any{value}
			}
		} else {
			valueList, exists := routeProxyRuleSet[name]
			if exists {
				if valueList, ok := valueList.([]any); ok {
					routeProxyRuleSet[name] = append(valueList, value)
				}
			} else {
				routeProxyRuleSet[name] = []any{value}
			}
		}
	}
	return nil
}

// 转换节点
func (c *Converter) convertOutbounds(clash *Clash) error {
	outboundNames := make([]string, 0)
	c.tmpl.Outbounds = make([]map[string]any, 0)
	for _, p := range clash.Proxies {
		ob := make(map[string]any)
		switch p.Type {
		case "ss":
			ob["type"] = "shadowsocks"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["method"] = p.Cipher
			ob["password"] = p.Password
			if p.Udp {
				ob["network"] = "udp"
			} else {
				ob["network"] = "tcp"
			}
		case "trojan":
			ob["type"] = "trojan"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["password"] = p.Password
			tls := make(map[string]any)
			tls["enabled"] = true
			tls["insecure"] = p.SkipCertVerify
			tls["server_name"] = p.Sni
			ob["tls"] = tls

			if p.Udp {
				ob["network"] = "udp"
			} else {
				ob["network"] = "tcp"
			}
		case "anytls":
			ob["type"] = "anytls"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["password"] = p.Password
			tls := make(map[string]any)
			tls["enabled"] = true
			tls["insecure"] = p.SkipCertVerify
			tls["server_name"] = p.Sni
			tls["alpn"] = p.Alpn
			utls := make(map[string]any)
			utls["enabled"] = true
			utls["fingerprint"] = p.ClientFingerprint
			tls["utls"] = utls

			ob["tls"] = tls
		default:
			fmt.Printf("unsupport protocol: %v\n", p.Type)
			continue
		}
		c.tmpl.Outbounds = append(c.tmpl.Outbounds, ob)
		outboundNames = append(outboundNames, p.Name)
	}

	c.tmpl.Outbounds = append(c.tmpl.Outbounds, map[string]any{
		"type":                        "selector",
		"tag":                         c.custom.NodeSelectionGroupName,
		"interrupt_exist_connections": true,
		"outbounds":                   append([]string{c.custom.AutoSelectionGroupName}, outboundNames...),
	})

	c.tmpl.Outbounds = append(c.tmpl.Outbounds, map[string]any{
		"type":                        "urltest",
		"tag":                         c.custom.AutoSelectionGroupName,
		"interrupt_exist_connections": true,
		"interval":                    "10m",
		"outbounds":                   outboundNames,
	})

	for _, s := range c.custom.OutboundSelectors {
		ob := map[string]any{
			"type":                        s.SelectorType,
			"tag":                         s.Tag,
			"interrupt_exist_connections": true,
			"outbounds": append(
				filterOutbound(outboundNames, s.Keywords),
				c.custom.DirectGroupName,
				c.custom.AutoSelectionGroupName,
				c.custom.NodeSelectionGroupName,
			),
		}
		if s.DefaultOutbound != "" {
			ob["default"] = s.DefaultOutbound
		}
		c.tmpl.Outbounds = append(c.tmpl.Outbounds, ob)
	}

	c.tmpl.Outbounds = append(c.tmpl.Outbounds, map[string]any{
		"type": "direct",
		"tag":  c.custom.DirectGroupName,
	})

	c.tmpl.Outbounds = append(c.tmpl.Outbounds, map[string]any{
		"type":                        "selector",
		"tag":                         c.custom.EscapeGroupName,
		"interrupt_exist_connections": true,
		"outbounds":                   []string{c.custom.NodeSelectionGroupName, c.custom.DirectGroupName},
		"default":                     c.custom.NodeSelectionGroupName,
	})
	return nil
}

func isDirect(outbound string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(outbound), keyword) {
			return true
		}
	}
	return false
}

func filterOutbound(outboundNames []string, keywords []string) []string {
	outbounds := make([]string, 0)
	if len(keywords) == 0 {
		return outbounds
	}
	for _, name := range outboundNames {
		for _, keyword := range keywords {
			if strings.Contains(name, keyword) {
				outbounds = append(outbounds, name)
			}
		}
	}
	return outbounds
}

func ruleType(clashRule string) (string, error) {
	switch clashRule {
	case "DOMAIN":
		return "domain", nil
	case "DOMAIN-SUFFIX":
		return "domain_suffix", nil
	case "DOMAIN-KEYWORD":
		return "domain_keyword", nil
	case "IP-CIDR", "IP-CIDR6":
		return "ip_cidr", nil
	case "PROCESS-NAME":
		return "process_name", nil
	default:
		return "", fmt.Errorf("unsupport condition name: %v\n", clashRule)
	}
}

func LoadSingboxFromPath(singboxPath string) (*SingBox, error) {
	data, err := os.ReadFile(singboxPath)
	if err != nil {
		return nil, fmt.Errorf("read %s error:\n\t%w", singboxPath, err)
	}
	sb := &SingBox{}
	if err := json.Unmarshal(data, sb); err != nil {
		return nil, fmt.Errorf("unmarshal json file %s error: \n\t%w", singboxPath, err)
	}
	return sb, nil
}
