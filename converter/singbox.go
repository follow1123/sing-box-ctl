package converter

type SingBox struct {
	Custom       *Custom          `json:"custom,omitempty"`
	Log          *Log             `json:"log,omitempty"`
	Experimental *Experimental    `json:"experimental,omitempty"`
	DNS          *DNS             `json:"dns,omitempty"`
	Inbounds     []map[string]any `json:"inbounds,omitempty"`
	Outbounds    []map[string]any `json:"outbounds,omitempty"`
	Route        *Route           `json:"route,omitempty"`
}

type Custom struct {
	// 默认使用的入站，模板内会把所有类型的 inbound 都填上，但是具体只使用一个
	DefaultInboundIndex int `json:"default_inbound_index"`

	NodeSelectionGroupName string `json:"node_selection_group_name"`
	AutoSelectionGroupName string `json:"auto_selection_group_name"`
	DirectGroupName        string `json:"direct_group_name"`
	EscapeGroupName        string `json:"escape_group_name"`

	DirectDNSServer           string            `json:"direct_dns_server,omitempty"`
	ProxyDNSServer            string            `json:"proxy_dns_server,omitempty"`
	DirectRuleKeywords        []string          `json:"direct_rule_keywords,omitempty"`
	DirectRuleSetIndexInDNS   int               `json:"direct_ruleset_index_in_dns,omitempty"`
	ProxyRuleSetIndexInDNS    int               `json:"proxy_ruleset_index_in_dns,omitempty"`
	DirectRuleSetIndexInRoute int               `json:"direct_ruleset_index_in_route,omitempty"`
	ProxyRuleSetIndexInRoute  int               `json:"proxy_ruleset_index_in_route,omitempty"`
	OutboundSelectors         []CustomSelectors `json:"selectors,omitempty"`
}

type CustomSelectors struct {
	Tag             string   `json:"tag"`
	SelectorType    string   `json:"type"`
	Keywords        []string `json:"keywords,omitempty"`
	DefaultOutbound string   `json:"default,omitempty"`
}

type Experimental struct {
	ClashAPI  *ClashAPI      `json:"clash_api,omitempty"`
	CacheFile map[string]any `json:"cache_file,omitempty"`
	V2rayApi  map[string]any `json:"v2ray_api,omitempty"`
}

type ClashAPI struct {
	ExternalController               string   `json:"external_controller,omitempty"`
	ExternalUI                       string   `json:"external_ui,omitempty"`
	ExternalUIDownloadUrl            string   `json:"external_ui_download_url,omitempty"`
	ExternalUIDownloadDetour         string   `json:"external_ui_download_detour,omitempty"`
	Secret                           string   `json:"secret,omitempty"`
	DefaultMode                      string   `json:"default_mode,omitempty"`
	AccessControlAllowOrigin         []string `json:"access_control_allow_origin,omitempty"`
	AccessControlAllowPrivateNetwork bool     `json:"access_control_allow_private_network,omitempty"`
}

type DNS struct {
	Servers          []map[string]any `json:"servers,omitempty"`
	Rules            []map[string]any `json:"rules,omitempty"`
	Final            string           `json:"final,omitempty"`
	Strategy         string           `json:"strategy,omitempty"`
	DisableCache     bool             `json:"disable_cache,omitempty"`
	DisableExpire    bool             `json:"disable_expire,omitempty"`
	IndependentCache bool             `json:"independent_cache,omitempty"`
	CacheCapacity    int              `json:"cache_capacity,omitempty"`
	Optimistic       any              `json:"optimistic,omitempty"`
	ReverseMapping   bool             `json:"reverse_mapping,omitempty"`
	ClientSubnet     string           `json:"client_subnet,omitempty"`
}

type Route struct {
	Geoip                      map[string]any   `json:"geoip,omitempty"`
	Geosite                    map[string]any   `json:"geosite,omitempty"`
	Rules                      []map[string]any `json:"rules,omitempty"`
	RuleSet                    []map[string]any `json:"rule_set,omitempty"`
	Final                      string           `json:"final,omitempty"`
	AutoDetectInterface        bool             `json:"auto_detect_interface,omitempty"`
	OverrideAndroidVpn         bool             `json:"override_android_vpn,omitempty"`
	DefaultInterface           string           `json:"default_interface,omitempty"`
	DefaultMark                int              `json:"default_mark,omitempty"`
	FindProcess                bool             `json:"find_process,omitempty"`
	FindNeighbor               bool             `json:"find_neighbor,omitempty"`
	DhcpLeaseFiles             []string         `json:"dhcp_lease_files,omitempty"`
	DefaultDomainResolver      string           `json:"default_domain_resolver,omitempty"`
	NetworkStrategy            string           `json:"network_strategy,omitempty"`
	DefaultNetworkType         string           `json:"default_network_type,omitempty"`
	DefaultFallbackNetworkType string           `json:"default_fallback_network_type,omitempty"`
	DefaultFallbackDelay       string           `json:"default_fallback_delay,omitempty"`
}

type Log struct {
	Disabled  bool   `json:"disabled,omitempty"`
	Level     string `json:"level,omitempty"`
	Output    string `json:"output,omitempty"`
	Timestamp bool   `json:"timestamp,omitempty"`
}
