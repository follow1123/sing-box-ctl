package converter

import (
	"fmt"
	"slices"
	"strings"
)

func Convert(clash *Clash, tmpl *SingBox) (*SingBox, error) {
	custom := tmpl.Custom
	// 清空自定义配置，防止生成到最终配置内
	tmpl.Custom = nil
	if err := convertOutbounds(clash, tmpl, custom); err != nil {
		return nil, fmt.Errorf("convert clash proxy to sing-box outbound error:\n\t%w", err)
	}
	if err := convertRules(clash, tmpl, custom); err != nil {
		return nil, fmt.Errorf("convert clash rule to sing-box rule error:\n\t%w", err)
	}
	tmpl.Inbounds = []map[string]any{tmpl.Inbounds[custom.DefaultInboundIndex]}
	return tmpl, nil
}

func convertRules(clash *Clash, tmpl *SingBox, custom *Custom) error {
	tmpl.DNS.Rules = slices.Insert(tmpl.DNS.Rules, custom.DirectRuleSetIndexInDNS, map[string]any{
		"rule_set": "providers_default_direct_rules", "server": custom.DirectDNSServer,
	})
	tmpl.DNS.Rules = slices.Insert(tmpl.DNS.Rules, custom.ProxyRuleSetIndexInDNS, map[string]any{
		"rule_set": "providers_default_proxy_rules", "server": custom.ProxyDNSServer,
	})

	routeDirectRuleSet := make(map[string]any, 0)
	routeProxyRuleSet := make(map[string]any, 0)

	tmpl.Route.Rules = slices.Insert(tmpl.Route.Rules, custom.DirectRuleSetIndexInRoute, map[string]any{
		"rule_set": "providers_default_direct_rules", "outbound": "直连",
	})
	tmpl.Route.Rules = slices.Insert(tmpl.Route.Rules, custom.ProxyRuleSetIndexInRoute, map[string]any{
		"rule_set": "providers_default_proxy_rules", "outbound": "节点选择",
	})
	tmpl.Route.RuleSet = append(
		tmpl.Route.RuleSet,
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
		if isDirect(outbound, custom.DirectRuleKeywords) {
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
func convertOutbounds(clash *Clash, sb *SingBox, custom *Custom) error {
	outboundNames := make([]string, 0)
	sb.Outbounds = make([]map[string]any, 0)
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
		sb.Outbounds = append(sb.Outbounds, ob)
		outboundNames = append(outboundNames, p.Name)
	}

	sb.Outbounds = append(sb.Outbounds, map[string]any{
		"type":                        "selector",
		"tag":                         custom.NodeSelectionGroupName,
		"interrupt_exist_connections": true,
		"outbounds":                   append([]string{custom.AutoSelectionGroupName}, outboundNames...),
	})

	sb.Outbounds = append(sb.Outbounds, map[string]any{
		"type":                        "urltest",
		"tag":                         custom.AutoSelectionGroupName,
		"interrupt_exist_connections": true,
		"interval":                    "10m",
		"outbounds":                   outboundNames,
	})

	for _, s := range custom.OutboundSelectors {
		ob := map[string]any{
			"type":                        s.SelectorType,
			"tag":                         s.Tag,
			"interrupt_exist_connections": true,
			"outbounds": append(
				filterOutbound(outboundNames, s.Keywords),
				custom.DirectGroupName,
				custom.AutoSelectionGroupName,
				custom.NodeSelectionGroupName,
			),
		}
		if s.DefaultOutbound != "" {
			ob["default"] = s.DefaultOutbound
		}
		sb.Outbounds = append(sb.Outbounds, ob)
	}

	sb.Outbounds = append(sb.Outbounds, map[string]any{
		"type": "direct",
		"tag":  custom.DirectGroupName,
	})

	sb.Outbounds = append(sb.Outbounds, map[string]any{
		"type":                        "selector",
		"tag":                         custom.EscapeGroupName,
		"interrupt_exist_connections": true,
		"outbounds":                   []string{custom.NodeSelectionGroupName, custom.DirectGroupName},
		"default":                     custom.NodeSelectionGroupName,
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
