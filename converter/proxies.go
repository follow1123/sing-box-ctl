package converter

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// ParseClashProxies 解析 clash 订阅，仅取出其中的节点列表（忽略 rules、
// proxy-groups、proxy-providers 等其它内容）。订阅文件最核心的就是节点，
// 入库/精简时只保留这部分即可大幅缩小体积。
func ParseClashProxies(clashData []byte) ([]Proxy, error) {
	clash := &Clash{}
	if err := yaml.Unmarshal(clashData, clash); err != nil {
		return nil, fmt.Errorf("unmarshal clash yaml config error:\n\t%w", err)
	}
	if len(clash.Proxies) == 0 {
		return nil, fmt.Errorf("no proxies in subscription")
	}
	return clash.Proxies, nil
}

// ExtractProxiesContent 解析订阅后仅保留节点，重新序列化为精简的 clash 片段
// （只含 proxies，可继续被 ParseClashProxies / Convert 消费）。
func ExtractProxiesContent(clashData []byte) ([]byte, error) {
	proxies, err := ParseClashProxies(clashData)
	if err != nil {
		return nil, err
	}
	out, err := yaml.Marshal(map[string]any{"proxies": proxies})
	if err != nil {
		return nil, fmt.Errorf("marshal proxies error:\n\t%w", err)
	}
	return out, nil
}
