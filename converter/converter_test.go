package converter_test

import (
	"fmt"
	"testing"

	"github.com/follow1123/sing-box-ctl/config"
	"github.com/follow1123/sing-box-ctl/converter"
	"github.com/follow1123/sing-box-ctl/settings"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestConvertSuccess(t *testing.T) {
	data := []byte(`
port: 7890
socks-port: 7891
redir-port: 7892
tproxy-port: 7893
mixed-port: 7894
allow-lan: true
bind-address: '*'
mode: rule
ipv6: false
log-level: info
external-controller: '0.0.0.0:9090'
dns:
  enable: true
  ipv6: false
  prefer-h3: true
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  use-hosts: true
  default-nameserver: [180.184.1.1, 119.29.29.29, 223.5.5.5]
  nameserver:
    - https://dns.alidns.com/dns-query
    - https://120.53.53.53/dns-query
    - https://doh.pub/dns-query
  fake-ip-filter:
    - "+.lan"
    - "+.local"
    - "+.market.xiaomi.com"
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
    #alpn:
    # - h2
    # - http/1.1
    skip-cert-verify: false
proxy-groups:

rules:
- IP-CIDR,102.198.138.0/18,🎯 直连,no-resolve
- IP-CIDR,113.198.180.0/19,🎯 直连,no-resolve
- IP-CIDR,213.199.199.0/22,🎯 直连,no-resolve
- GEOIP,CN,🎯 直连
- MATCH,🐟 漏网之鱼`)

	conf, err := config.Default()
	assert.NoError(t, err)
	sbTmpl, err := settings.LoadConfigFromPath(conf.SingBoxTmplConfigPath())
	assert.NoError(t, err)

	c := &converter.Clash{}
	err = yaml.Unmarshal(data, c)
	assert.NoError(t, err)

	sb, err := converter.Convert(c, sbTmpl)
	assert.NoError(t, err)
	fmt.Printf("sb.Outbounds: %v\n", sb.Outbounds)
	assert.Contains(t, sb.Outbounds[0]["tag"], "aaa")
	assert.Contains(t, sb.Outbounds[1]["tag"], "bbb")
}

func TestConvertFailure(t *testing.T) {
}
