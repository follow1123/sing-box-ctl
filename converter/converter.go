package converter

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

//go:embed config.json.tmpl
var singBoxConfigTemplate string

func Convert(data []byte) ([]byte, error) {
	cc := &ClashConfig{}
	if err := yaml.Unmarshal(data, cc); err != nil {
		return nil, fmt.Errorf("unmarshal clash config error: \n\t%w", err)
	}
	sbc := clashToSingBox(cc)

	tmpl := template.New("sing-box-config-tmpl").Funcs(template.FuncMap{
		"nodeFilter": nodeFilter,
	})
	tmpl, err := tmpl.Parse(singBoxConfigTemplate)
	if err != nil {
		return nil, fmt.Errorf("load tempalte error: \n\t%w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sbc); err != nil {
		return nil, fmt.Errorf("execute tempalte error: \n\t%w", err)
	}

	return buf.Bytes(), nil
}

func nodeFilter(outbounds []Outbound, keys string) []string {
	keyArr := strings.Split(keys, "|")
	result := make([]string, 0)
	for _, outbound := range outbounds {
		for _, k := range keyArr {
			if strings.Contains(outbound.Tag, k) {
				result = append(result, outbound.Tag)
				break
			}
		}
	}
	return result
}

func clashToSingBox(cc *ClashConfig) *SingBoxConfig {
	sbc := &SingBoxConfig{
		Outbounds:     make([]Outbound, 0),
		Rules:         make([]Rule, 0),
		InlineRuleSet: make([]InlineRuleSet, 0),
	}
	convertProxies(cc, sbc)
	convertRules(cc, sbc)
	return sbc
}

// 转换协议
func convertProxies(cc *ClashConfig, sbc *SingBoxConfig) {
	for _, p := range cc.Proxies {
		var ob Outbound

		network := "tcp"
		if p.Udp {
			network = "udp"
		}
		switch p.Type {
		case "ss":
			ob = Outbound{
				Type: "shadowsocks",
				Tag:  p.Name,
				Protocol: Shadowsocks{
					Server:     p.Server,
					ServerPort: p.Port,
					Method:     p.Cipher,
					Password:   p.Password,
					Network:    network,
				},
			}
		case "trojan":
			ob = Outbound{
				Type: "trojan",
				Tag:  p.Name,
				Protocol: Trojan{
					Server:     p.Server,
					ServerPort: p.Port,
					Password:   p.Password,
					Network:    "tcp",
					Tls: Tls{
						Enabled:    !p.SkipCertVerify,
						ServerName: p.Sni,
					},
				},
			}
		default:
			log.Printf("unsupport protocol: %v\n", p.Type)
			continue
		}
		sbc.Outbounds = append(sbc.Outbounds, ob)
	}
}

// 转换规则
func convertRules(cc *ClashConfig, sbc *SingBoxConfig) {
	currentOutbound := ""
	ruleIdx := -1
	for i, r := range cc.Rules {
		items := strings.Split(r, ",")

		if len(items) < 3 {
			log.Printf("ignore rule：%v\n", r)
			continue
		}

		name := ruleType(items[0])
		if name == "" {
			log.Printf("unsupport condition name: %v\n", items[0])
			continue
		}
		value := items[1]
		outbound := items[2]
		if currentOutbound != outbound || i == len(cc.Rules)-1 {
			currentOutbound = outbound
			ruleIdx++
			ruleSetName := fmt.Sprintf("providers-builtin-rule-%v", ruleIdx+1)
			if strings.Contains(outbound, "直连") || strings.Contains(strings.ToLower(outbound), "direct") {
				outbound = "直连"
			} else if strings.Contains(outbound, "选择") || strings.Contains(strings.ToLower(outbound), "select") {
				outbound = "节点选择"
			}
			sbc.Rules = append(sbc.Rules, Rule{
				RuleSet:  ruleSetName,
				Outbound: outbound,
			})
			headlessRule := &HeadlessRule{
				Conditions: make([]RuleCondition, 0),
			}
			headlessRule.AddCondition(name, value)
			sbc.InlineRuleSet = append(sbc.InlineRuleSet, InlineRuleSet{
				Tag:          ruleSetName,
				HeadlessRule: headlessRule,
			})
			continue
		}
		sbc.InlineRuleSet[ruleIdx].HeadlessRule.AddCondition(name, value)
	}
}

func ruleType(clashRule string) string {
	switch clashRule {
	case "DOMAIN":
		return "domain"
	case "DOMAIN-SUFFIX":
		return "domain_suffix"
	case "DOMAIN-KEYWORD":
		return "domain_keyword"
	case "IP-CIDR", "IP-CIDR6":
		return "ip_cidr"
	case "PROCESS-NAME":
		return "process_name"
	default:
		return ""
	}
}
