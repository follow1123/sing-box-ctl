package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type Converter struct {
	tmpl *SingBox
}

func New(singboxTemplateConfigPath string) (*Converter, error) {
	tmplConf, err := LoadSingboxFromPath(singboxTemplateConfigPath)
	if err != nil {
		return nil, err
	}
	return &Converter{tmpl: tmplConf}, nil
}

// NodesFromClash 解析 clash 订阅，仅提取其中的节点并转换为 sing-box outbounds
// （不含订阅的 rules/groups 等）。供订阅入库前调用——入库即最终节点形态，
// 后续生成配置不再依赖订阅原文。
func NodesFromClash(clashData []byte) ([]map[string]any, error) {
	clash := &Clash{}
	if err := yaml.Unmarshal(clashData, clash); err != nil {
		return nil, fmt.Errorf("unmarshal clash yaml config error:\n\t%w", err)
	}
	nodes, names := convertNodes(clash.Proxies)
	if len(names) == 0 {
		return nil, fmt.Errorf("no valid proxies in subscription")
	}
	return nodes, nil
}

// Convert 将 clash 订阅转换为 sing-box 配置：
//   - 只解析订阅中的节点（proxies），不解析订阅规则（rules）
//   - 模板的 dns/route/rule_set、inbounds、services 等原样保留
//   - inbounds 的启停由 settings 根据 URL 参数控制，这里不做截断
//   - outbounds 中 tag 含 @ 表达式的组会被填充节点（见 resolveOutboundExpr）
func (c *Converter) Convert(clashData []byte) (*SingBox, error) {
	nodes, err := NodesFromClash(clashData)
	if err != nil {
		return nil, err
	}
	return c.Build(nodes)
}

// Build 以已加载的模板为基础，把订阅节点 outbounds 前置为最终 outbounds，
// 并解析模板组 tag 中的 @ 表达式（见 resolveOutboundExpr）。
// 不改动接收者，每次返回全新配置。
func (c *Converter) Build(nodes []map[string]any) (*SingBox, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxies in subscription")
	}
	tmpl, err := cloneSingBox(c.tmpl)
	if err != nil {
		return nil, err
	}
	nodeNames := make([]string, 0, len(nodes))
	for _, ob := range nodes {
		if tag, ok := ob["tag"].(string); ok {
			nodeNames = append(nodeNames, tag)
		}
	}
	for _, ob := range tmpl.Outbounds {
		resolveOutboundExpr(ob, nodeNames)
	}
	// 最终 outbounds = 订阅节点 + 模板分组
	tmpl.Outbounds = append(nodes, tmpl.Outbounds...)
	return tmpl, nil
}

// cloneSingBox 深拷贝模板，保证 Build 不累积修改接收者的模板
func cloneSingBox(sb *SingBox) (*SingBox, error) {
	data, err := json.Marshal(sb)
	if err != nil {
		return nil, fmt.Errorf("clone template error:\n\t%w", err)
	}
	clone := &SingBox{}
	if err := json.Unmarshal(data, clone); err != nil {
		return nil, fmt.Errorf("clone template error:\n\t%w", err)
	}
	return clone, nil
}

// convertNodes 将 clash 节点转换为 sing-box outbound
func convertNodes(proxies []Proxy) ([]map[string]any, []string) {
	outbounds := make([]map[string]any, 0)
	names := make([]string, 0)
	for _, p := range proxies {
		ob := make(map[string]any)
		switch p.Type {
		case "ss":
			ob["type"] = "shadowsocks"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["method"] = p.Cipher
			ob["password"] = p.Password
			setNetwork(ob, p)
		case "vmess":
			ob["type"] = "vmess"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["uuid"] = p.Uuid
			if p.AlterId > 0 {
				ob["alter_id"] = p.AlterId
			}
			ob["security"] = p.Cipher
			if ob["security"] == "" {
				ob["security"] = "auto"
			}
			if p.Tls {
				ob["tls"] = buildTls(p)
			}
			if p.Network == "ws" || p.WsOpts != nil {
				ob["transport"] = buildWsTransport(p)
			}
			setNetwork(ob, p)
		case "vless":
			ob["type"] = "vless"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["uuid"] = p.Uuid
			if p.Flow != "" {
				ob["flow"] = p.Flow
			}
			if p.Tls {
				ob["tls"] = buildTls(p)
			}
			if p.Network == "ws" || p.WsOpts != nil {
				ob["transport"] = buildWsTransport(p)
			}
			setNetwork(ob, p)
		case "trojan":
			ob["type"] = "trojan"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["password"] = p.Password
			ob["tls"] = buildTls(p)
			setNetwork(ob, p)
		case "anytls":
			ob["type"] = "anytls"
			ob["tag"] = p.Name
			ob["server"] = p.Server
			ob["server_port"] = p.Port
			ob["password"] = p.Password
			ob["tls"] = buildTls(p)
		default:
			fmt.Printf("unsupport protocol: %v\n", p.Type)
			continue
		}
		outbounds = append(outbounds, ob)
		names = append(names, p.Name)
	}
	return outbounds, names
}

// buildTls 构造 tls 配置
func buildTls(p Proxy) map[string]any {
	tls := map[string]any{
		"enabled":     true,
		"insecure":    p.SkipCertVerify,
		"server_name": p.Sni,
	}
	if len(p.Alpn) > 0 {
		tls["alpn"] = p.Alpn
	}
	if p.ClientFingerprint != "" {
		tls["utls"] = map[string]any{"enabled": true, "fingerprint": p.ClientFingerprint}
	}
	return tls
}

// buildWsTransport 构造 ws 传输层配置
func buildWsTransport(p Proxy) map[string]any {
	ws := map[string]any{"type": "ws"}
	if p.WsOpts != nil {
		if p.WsOpts.Path != "" {
			ws["path"] = p.WsOpts.Path
		}
		if len(p.WsOpts.Headers) > 0 {
			ws["headers"] = p.WsOpts.Headers
		}
	}
	return ws
}

// setNetwork 根据 udp 标记设置 network 字段
func setNetwork(ob map[string]any, p Proxy) {
	if p.Udp {
		ob["network"] = "udp"
	} else {
		ob["network"] = "tcp"
	}
}

// resolveOutboundExpr 处理 outbound tag 中的 @ 表达式：
//   - "组名@all"              -> 填充全部节点
//   - "组名@keywords=台湾,tw"  -> 填充包含任一关键词的节点
//   - "组名@exclude=美国,mg"   -> 填充不包含任一关键词的节点
//   - 无 @ 的组原样保留
//
// 筛选出的节点 append 到该组已有的 outbounds 之后，tag 还原为组名
func resolveOutboundExpr(ob map[string]any, nodeNames []string) {
	tag, _ := ob["tag"].(string)
	parts := strings.SplitN(tag, "@", 2)
	if len(parts) != 2 {
		return
	}
	groupName, expr := parts[0], parts[1]
	ob["tag"] = groupName

	var matched []string
	switch {
	case expr == "all":
		matched = nodeNames
	case strings.HasPrefix(expr, "keywords="):
		keywords := strings.Split(strings.TrimPrefix(expr, "keywords="), ",")
		matched = filterNodes(nodeNames, keywords, true)
	case strings.HasPrefix(expr, "exclude="):
		keywords := strings.Split(strings.TrimPrefix(expr, "exclude="), ",")
		matched = filterNodes(nodeNames, keywords, false)
	default:
		// 未知表达式，忽略（保持 tag 原样？不，已拆分了，保留组名）
		return
	}

	list, _ := ob["outbounds"].([]any)
	for _, name := range matched {
		list = append(list, name)
	}
	ob["outbounds"] = list
}

// filterNodes 按关键词过滤节点名（或语义：包含任一关键词即匹配）
func filterNodes(nodeNames []string, keywords []string, include bool) []string {
	result := make([]string, 0)
	for _, name := range nodeNames {
		matched := false
		for _, kw := range keywords {
			if kw != "" && strings.Contains(name, kw) {
				matched = true
				break
			}
		}
		if matched == include {
			result = append(result, name)
		}
	}
	return result
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
