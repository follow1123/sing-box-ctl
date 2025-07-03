package singbox

import (
	"fmt"
	"strings"
)

type ProxyMode string

const (
	Mixed   ProxyMode = "mixed"
	MixedSP ProxyMode = "mixed-sp"
	TUN     ProxyMode = "tun"
)

type Opt func(*Options)

func WithWebUI(enable bool) Opt {
	return func(o *Options) {
		o.WebUIEnabled = enable
	}
}

func WithAndroid(enable bool) Opt {
	return func(o *Options) {
		o.Android = enable
	}
}

func WithWebUIAddr(addr string) Opt {
	return func(o *Options) {
		o.WebUIEnabled = true
		o.WebUIAddr = addr
	}
}

func WithMode(mode ProxyMode) Opt {
	return func(o *Options) {
		o.Mode = mode
	}
}

type Options struct {
	WebUIEnabled bool
	WebUIAddr    string
	Mode         ProxyMode
	Android      bool
}

type PartialConfig struct {
	NodeSelect    string
	Escapes       string
	NodeNames     []string
	Outbounds     []*Outbound
	Rules         []*Rule
	InlineRuleSet []*InlineRuleSet
}

func NewPartialConfig() *PartialConfig {
	return &PartialConfig{
		NodeNames:     make([]string, 0),
		Outbounds:     make([]*Outbound, 0),
		Rules:         make([]*Rule, 0),
		InlineRuleSet: make([]*InlineRuleSet, 0),
	}
}

func (pc *PartialConfig) SetNodeSelect(name string) {
	if pc.NodeSelect == "" &&
		(strings.Contains(name, "节点选择") || strings.Contains(name, "选择")) {
		pc.NodeSelect = name
	}
}

func (pc *PartialConfig) SetEscapes(name string) {
	if pc.Escapes == "" &&
		(strings.Contains(name, "漏网之鱼") || strings.Contains(name, "漏网")) {
		pc.Escapes = name
	}
}

type TemplateData struct {
	Opts *Options
	Pc   *PartialConfig
}

func StrToProxyMode(mode string) (ProxyMode, error) {
	switch mode {
	case string(Mixed):
		return Mixed, nil
	case string(MixedSP):
		return MixedSP, nil
	case string(TUN):
		return TUN, nil
	default:
		return "", fmt.Errorf("invalid ProxyMode: %s, available modes: [mixed|mixed-sp|tun]", mode)
	}
}

type Outbound struct {
	Type string
	Tag  string
	Data any
}

type Shadowsocks struct {
	Server     string
	ServerPort int
	Password   string
	Network    string
	Method     string
}

type Tls struct {
	Enabled    bool
	ServerName string
}

type Trojan struct {
	Server     string
	ServerPort int
	Password   string
	Network    string
	Tls        Tls
}

type Selector struct {
	Outbounds                 []string
	InterruptExistConnections bool
}

type UrlTest struct {
	Outbounds                 []string
	InterruptExistConnections bool
	Url                       string
	Interval                  string
	Tolerance                 int
}

type Rule struct {
	RuleSet  string
	Outbound string
}

type RuleCondition struct {
	Name  string
	Value []string
}

type HeadlessRule struct {
	Conditions []RuleCondition
}

func (hr *HeadlessRule) AddCondition(name string, value string) {
	for i := range hr.Conditions {
		if hr.Conditions[i].Name == name {
			hr.Conditions[i].Value = append(hr.Conditions[i].Value, value)
			return
		}
	}
	hr.Conditions = append(hr.Conditions, RuleCondition{
		Name:  name,
		Value: []string{value},
	})
}

type InlineRuleSet struct {
	Tag          string
	HeadlessRule *HeadlessRule
}
