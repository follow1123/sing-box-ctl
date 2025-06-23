package clash

import (
	"fmt"
	"math"
	"strings"

	"github.com/follow1123/sing-box-ctl/logger"
	"github.com/follow1123/sing-box-ctl/singbox"
)

func convertRules(log logger.Logger, cc *ClashConfig, pc *singbox.PartialConfig) {
	currentProxy := ""
	ruleIdx := -1
	for i, r := range cc.Rules {
		items := strings.Split(r, ",")

		if len(items) < 3 {
			log.Debug("ignore rule：%v", r)
			continue
		}

		name := ruleType(items[0])
		if name == "" {
			log.Warn("unsupport condition name: %v", items[0])
			continue
		}
		value := items[1]
		proxy := items[2]
		if currentProxy != proxy || i == len(cc.Rules)-1 {
			currentProxy = proxy
			ruleIdx++
			ruleSetName := fmt.Sprintf("subscription-builtin-rule-%v", ruleIdx+1)
			pc.Rules = append(pc.Rules, &singbox.Rule{
				RuleSet:  ruleSetName,
				Outbound: proxy,
			})
			headlessRule := &singbox.HeadlessRule{
				Conditions: make([]singbox.RuleCondition, 0),
			}
			headlessRule.AddCondition(name, value)
			pc.InlineRuleSet = append(pc.InlineRuleSet, &singbox.InlineRuleSet{
				Tag:          ruleSetName,
				HeadlessRule: headlessRule,
			})
			continue
		}
		pc.InlineRuleSet[ruleIdx].HeadlessRule.AddCondition(name, value)
	}
}

func convertProxies(log logger.Logger, cc *ClashConfig, pc *singbox.PartialConfig) {
	for _, p := range cc.Proxies {
		var ob singbox.Outbound

		network := "tcp"
		if p.Udp {
			network = "udp"
		}
		switch p.Type {
		case "ss":
			ob = singbox.Outbound{
				Type: "shadowsocks",
				Tag:  p.Name,
				Data: singbox.Shadowsocks{
					Server:     p.Server,
					ServerPort: p.Port,
					Method:     p.Cipher,
					Password:   p.Password,
					Network:    network,
				},
			}
		case "trojan":
			ob = singbox.Outbound{
				Type: "trojan",
				Tag:  p.Name,
				Data: singbox.Trojan{
					Server:     p.Server,
					ServerPort: p.Port,
					Password:   p.Password,
					Network:    "tcp",
					Tls: singbox.Tls{
						Enabled:    !p.SkipCertVerify,
						ServerName: p.Sni,
					},
				},
			}
		default:
			log.Warn("unsupport protocol: %v", p.Type)
			continue
		}
		pc.Outbounds = append(pc.Outbounds, &ob)
		pc.NodeNames = append(pc.NodeNames, ob.Tag)
	}
}

func convertProxyGroups(log logger.Logger, cc *ClashConfig, pc *singbox.PartialConfig) {
	for _, pg := range cc.ProxyGroups {
		var ob singbox.Outbound
		switch pg.Type {
		case "select":
			ob = singbox.Outbound{
				Type: "selector",
				Tag:  pg.Name,
				Data: singbox.Selector{
					Outbounds:                 pg.Proxies,
					InterruptExistConnections: false,
				},
			}
		case "url-test":
			var minutes = ""
			if pg.Interval > 60 {
				intervalMinutes := float32(pg.Interval) / 60.0
				minutes = fmt.Sprintf("%.0fm", math.Ceil(float64(intervalMinutes)))
			}
			ob = singbox.Outbound{
				Type: "urltest",
				Tag:  pg.Name,
				Data: singbox.UrlTest{
					Url:       pg.Url,
					Interval:  minutes,
					Tolerance: pg.Tolerance,
					Outbounds: pg.Proxies,
				},
			}
		default:
			log.Warn("unsupport protocol: %v", pg.Type)
			continue
		}
		pc.SetEscapes(ob.Tag)
		pc.SetNodeSelect(ob.Tag)
		pc.Outbounds = append(pc.Outbounds, &ob)
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
