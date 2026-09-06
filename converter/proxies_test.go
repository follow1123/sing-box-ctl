package converter_test

import (
	"strings"
	"testing"

	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 构造一个“杂讯较多”的 clash 订阅：节点之外还包含 proxy-groups/proxy-providers/
// 大量 rules/订阅头部注释，用于验证抠节点后这些内容都被丢弃。
const clashHead = `
# subscription update: 2026-09-01 10:00
# expire: 2026-12-31
proxies:
  - name: 节点-1
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-128-gcm
    password: secret
    udp: true
  - name: 节点-2
    type: vmess
    server: 2.2.2.2
    port: 443
    uuid: 11111111-2222-3333-4444-555555555555
    alterId: 0
    cipher: auto
    tls: true
    network: ws
    ws-opts:
      path: /path
      headers:
        Host: example.org
  - name: 节点-3
    type: trojan
    server: 3.3.3.3
    port: 8443
    password: token
    sni: node.example.org
    skip-cert-verify: true

proxy-providers:
  extra-provider:
    type: http
    url: "https://example.org/sub?token=xxxxx"
    interval: 86400
    path: ./providers/extra.yaml
    health-check:
      enable: true
      interval: 600
      url: http://www.gstatic.com/generate_204

proxy-groups:
  - name: 节点选择
    type: select
    proxies:
      - 节点-1
      - 节点-2
      - 节点-3
  - name: 自动选择
    type: url-test
    url: http://www.gstatic.com/generate_204
    interval: 300
    proxies:
      - 节点-1
      - 节点-2

rules:
`

// 大量规则行模拟真实订阅的“非节点内容”
var noisyClash = clashHead + strings.Repeat("  - DOMAIN-SUFFIX,noise-0001.example.com,节点选择\n", 150)

func TestParseClashProxies(t *testing.T) {
	proxies, err := converter.ParseClashProxies([]byte(noisyClash))
	require.NoError(t, err)
	require.Len(t, proxies, 3)

	// 类型与基本字段
	assert.Equal(t, "ss", proxies[0].Type)
	assert.Equal(t, "节点-1", proxies[0].Name)
	assert.Equal(t, "1.2.3.4", proxies[0].Server)
	assert.Equal(t, 8388, proxies[0].Port)
	assert.True(t, proxies[0].Udp)

	assert.Equal(t, "vmess", proxies[1].Type)
	assert.Equal(t, 443, proxies[1].Port)
	require.NotNil(t, proxies[1].WsOpts)
	assert.Equal(t, "/path", proxies[1].WsOpts.Path)
	assert.Equal(t, "example.org", proxies[1].WsOpts.Headers["Host"])
	assert.True(t, proxies[1].Tls)

	assert.Equal(t, "trojan", proxies[2].Type)
	assert.True(t, proxies[2].SkipCertVerify)
	assert.Equal(t, "node.example.org", proxies[2].Sni)
}

func TestExtractProxiesContent(t *testing.T) {
	out, err := converter.ExtractProxiesContent([]byte(noisyClash))
	require.NoError(t, err)

	text := string(out)
	// 其余部分被抠掉：不再包含 rules / proxy-groups / proxy-providers / 头部注释
	assert.NotContains(t, text, "rules:")
	assert.NotContains(t, text, "proxy-groups:")
	assert.NotContains(t, text, "proxy-providers:")
	assert.NotContains(t, text, "subscription update")
	assert.NotContains(t, text, "noise-")

	// 保留的节点数量与字段一致（可再次被解析）
	proxies, err := converter.ParseClashProxies(out)
	require.NoError(t, err)
	assert.Len(t, proxies, 3)
	assert.Equal(t, "节点-3", proxies[2].Name)

	// 体积明显缩小（本例杂讯远大于节点部分）
	assert.Less(t, len(out), len(noisyClash)/5, "抠出节点后体积应显著小于原始订阅")
}

func TestExtractProxiesContentErrors(t *testing.T) {
	// 没有节点
	_, err := converter.ExtractProxiesContent([]byte("proxies: []\nrules:\n  - MATCH,节点选择\n"))
	assert.ErrorContains(t, err, "no proxies")

	// 非法 yaml
	_, err = converter.ExtractProxiesContent([]byte("proxies: [\n  - broken"))
	assert.ErrorContains(t, err, "unmarshal clash yaml")
}

// 精简结果应能被 Convert 完整消费（作为后续入库/生成配置的输入）
func TestExtractProxiesContentFeedsConvert(t *testing.T) {
	out, err := converter.ExtractProxiesContent([]byte(noisyClash))
	require.NoError(t, err)

	conv, err := converter.New(writeTmplConfig(t))
	require.NoError(t, err)
	sb, err := conv.Convert(out)
	require.NoError(t, err)

	// 三个节点进最终 outbounds（模板自身 4 组 + 直连 = 5，节点 +3）
	assert.Equal(t, 3, len(sb.Outbounds)-5)
}
