package converter

type SingBoxConfig struct {
	Outbounds     []Outbound
	Rules         []Rule
	InlineRuleSet []InlineRuleSet
}

type Outbound struct {
	Type     string
	Tag      string
	Protocol any
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
