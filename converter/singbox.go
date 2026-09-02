package converter

type SingBox struct {
	Log          *Log             `json:"log,omitempty"`
	Experimental *Experimental    `json:"experimental,omitempty"`
	HttpClients  []map[string]any `json:"http_clients,omitempty"`
	Services     []map[string]any `json:"services,omitempty"`
	DNS          *DNS             `json:"dns,omitempty"`
	Inbounds     []map[string]any `json:"inbounds,omitempty"`
	Outbounds    []map[string]any `json:"outbounds,omitempty"`
	Route        *Route           `json:"route,omitempty"`
}

type Experimental struct {
	CacheFile map[string]any `json:"cache_file,omitempty"`
	V2rayApi  map[string]any `json:"v2ray_api,omitempty"`
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
	DefaultHttpClient         string           `json:"default_http_client,omitempty"`
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
