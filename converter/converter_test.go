package converter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/stretchr/testify/assert"
)

const tmplConfig = `{
  "custom": {
    "default_inbound_index": 0,
    "node_selection_group_name": "节点选择",
    "auto_selection_group_name": "自动选择",
    "direct_group_name": "直连",
    "escape_group_name": "漏网之鱼",
    "direct_dns_server": "dns-ali",
    "proxy_dns_server": "dns-google",
    "direct_rule_keywords": ["直连", "direct", "绕过代理"],
    "direct_ruleset_index_in_dns": 0,
    "proxy_ruleset_index_in_dns": 1,
    "direct_ruleset_index_in_route": 0,
    "proxy_ruleset_index_in_route": 1,
    "selectors": []
  },
  "dns": {
    "servers": [],
    "rules": []
  },
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.1",
      "listen_port": 7899
    }
  ],
  "outbounds": [],
  "route": {
    "rules": [],
    "rule_set": []
  }
}`

func writeTmplConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tmpl.json")
	if err := os.WriteFile(path, []byte(tmplConfig), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConvertSuccess(t *testing.T) {
	data := []byte(`
proxies:
  - name: "aaa"
    server: a.com
    port: 10229
    type: ss
    cipher: chacha20-ietf-poly1305
    password: "123456"

  - name: "bbb"
    server: b.com
    port: 12345
    type: trojan
    password: deaf3be4-7a94-32d0-8d54-d6610143755d
    sni: b.b.com
    skip-cert-verify: false

rules:
- IP-CIDR,102.198.138.0/18,🎯 直连,no-resolve
- IP-CIDR,113.198.180.0/19,🎯 直连,no-resolve
- IP-CIDR,213.199.199.0/22,🎯 直连,no-resolve
- GEOIP,CN,🎯 直连
- MATCH,🐟 漏网之鱼`)

	conv, err := converter.New(writeTmplConfig(t))
	assert.NoError(t, err)

	sb, err := conv.Convert(data)
	assert.NoError(t, err)
	fmt.Printf("sb.Outbounds: %v\n", sb.Outbounds)

	// 节点转换
	assert.Contains(t, sb.Outbounds[0]["tag"], "aaa")
	assert.Contains(t, sb.Outbounds[1]["tag"], "bbb")
	// 分组
	assert.Equal(t, "selector", sb.Outbounds[2]["type"])
	assert.Equal(t, "节点选择", sb.Outbounds[2]["tag"])
	assert.Equal(t, "urltest", sb.Outbounds[3]["type"])
	assert.Equal(t, "直连", sb.Outbounds[len(sb.Outbounds)-2]["tag"])
	assert.Equal(t, "漏网之鱼", sb.Outbounds[len(sb.Outbounds)-1]["tag"])
	// 规则转换
	assert.Equal(t, 2, len(sb.DNS.Rules))
	assert.Equal(t, 2, len(sb.Route.Rules))
	assert.Equal(t, 2, len(sb.Route.RuleSet))
	// 只保留默认 inbound
	assert.Equal(t, 1, len(sb.Inbounds))
	assert.Equal(t, "mixed", sb.Inbounds[0]["type"])
}

func TestConvertFailure(t *testing.T) {
	conv, err := converter.New(writeTmplConfig(t))
	assert.NoError(t, err)

	// 无效的 clash 配置
	_, err = conv.Convert([]byte("invalid: [yaml"))
	assert.Error(t, err)
}
