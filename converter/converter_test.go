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
    },
    {
      "type": "tun",
      "tag": "tun-in"
    }
  ],
  "outbounds": [
    {
      "tag": "节点选择@all",
      "type": "selector",
      "outbounds": ["自动选择"]
    },
    {
      "tag": "自动选择@all",
      "type": "urltest",
      "outbounds": []
    },
    {
      "tag": "直连",
      "type": "direct"
    },
    {
      "tag": "OPENAI@keywords=台湾,tw",
      "type": "selector",
      "outbounds": ["直连", "自动选择", "节点选择"],
      "default": "自动选择"
    },
    {
      "tag": "GITHUB@exclude=美国",
      "type": "selector",
      "outbounds": ["直连", "自动选择", "节点选择"],
      "default": "自动选择"
    }
  ],
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
  - name: "台湾-01"
    server: a.com
    port: 10229
    type: ss
    cipher: chacha20-ietf-poly1305
    password: "123456"

  - name: "美国-01"
    server: b.com
    port: 12345
    type: trojan
    password: deaf3be4-7a94-32d0-8d54-d6610143755d
    sni: b.b.com

  - name: "日本-01"
    server: c.com
    port: 443
    type: anytls
    password: "xxx"`)

	conv, err := converter.New(writeTmplConfig(t))
	assert.NoError(t, err)

	sb, err := conv.Convert(data)
	assert.NoError(t, err)
	fmt.Printf("sb.Outbounds: %v\n", sb.Outbounds)

	// 节点转换（最前三个是节点）
	assert.Equal(t, 3, len(sb.Outbounds) - 5)
	assert.Equal(t, "台湾-01", sb.Outbounds[0]["tag"])
	assert.Equal(t, "日本-01", sb.Outbounds[2]["tag"])

	// 按 tag 索引分组
	groups := make(map[string]map[string]any)
	for _, ob := range sb.Outbounds {
		groups[ob["tag"].(string)] = ob
	}

	// 节点选择：全部节点 + 自动选择
	nodeSel := groups["节点选择"]["outbounds"].([]any)
	assert.Equal(t, 4, len(nodeSel))
	assert.Equal(t, "自动选择", nodeSel[0])
	assert.Contains(t, nodeSel, "台湾-01")
	assert.Contains(t, nodeSel, "美国-01")
	assert.Contains(t, nodeSel, "日本-01")

	// 自动选择：全部节点
	autoSel := groups["自动选择"]["outbounds"].([]any)
	assert.Equal(t, 3, len(autoSel))

	// OPENAI：包含"台湾"关键词的节点 append 在兜底组后
	openai := groups["OPENAI"]["outbounds"].([]any)
	assert.Equal(t, 4, len(openai))
	assert.Equal(t, "直连", openai[0])
	assert.Contains(t, openai, "台湾-01")
	assert.NotContains(t, openai, "美国-01")
	assert.NotContains(t, openai, "日本-01")

	// GITHUB：排除"美国"的节点 append 在兜底组后
	github := groups["GITHUB"]["outbounds"].([]any)
	assert.Equal(t, 5, len(github))
	assert.Equal(t, "直连", github[0])
	assert.Contains(t, github, "台湾-01")
	assert.Contains(t, github, "日本-01")
	assert.NotContains(t, github, "美国-01")

	// 直连组原样
	assert.Equal(t, "direct", groups["直连"]["type"])

	// tag 已还原（无 @ 表达式）
	for _, ob := range sb.Outbounds {
		assert.NotContains(t, ob["tag"], "@")
	}

	// 只保留第一个 inbound
	assert.Equal(t, 1, len(sb.Inbounds))
	assert.Equal(t, "mixed", sb.Inbounds[0]["type"])

	// 规则原样（不解析订阅规则）
	assert.Equal(t, 0, len(sb.DNS.Rules))
	assert.Equal(t, 0, len(sb.Route.Rules))
}

func TestConvertFailure(t *testing.T) {
	conv, err := converter.New(writeTmplConfig(t))
	assert.NoError(t, err)

	// 无效的 clash 配置
	_, err = conv.Convert([]byte("invalid: [yaml"))
	assert.Error(t, err)

	// 无节点订阅
	_, err = conv.Convert([]byte("proxies: []\n"))
	assert.Error(t, err)
}
