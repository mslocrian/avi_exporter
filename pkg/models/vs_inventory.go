package models

type VsInventory struct {
	Count   int64  `json:"count"`
	Next    string `json:"next"`
	Results []VirtualServiceInventory
}

type VirtualServiceInventory struct {
	Config                     VirtualServiceConfig `json:"config"`
	Runtime                    interface{}          `json:"runtime"`
	UUID                       string               `json:"uuid"`
	HealthScore                interface{}          `json:"health_score"`
	Alert                      interface{}          `json:"alert"`
	Pools                      []string             `json:"pools"`
	PoolGroups                 []string             `json:"poolgroups"`
	AppProfileType             string               `json:"app_profile_type"`
	HasPoolWithRealtimeMetrics bool                 `json:"has_pool_with_realtime_metrics"`
	Faults                     interface{}          `json:"faults"`
	Metrics                    interface{}          `json:"metrics"`
}

type VirtualServiceConfig struct {
	Name              string                        `json:"name"`
	UUID              string                        `json:"uuid"`
	URL               string                        `json:"url"`
	Service           []VirtualServiceConfigService `json:"url"`
	PoolRef           string                        `json:"pool_ref"`
	FQDN              string                        `json:"fqdn"`
	Type              string                        `json:"type"`
	VHDomainName      string                        `json:"vh_domain_name"`
	TenantRef         string                        `json:"tenant_ref"`
	CloudRef          string                        `json:"cloud_ref"`
	SEGroupRef        string                        `json:"se_group_ref"`
	VRFContextRef     string                        `json:"vrf_context_ref"`
	Enabled           bool                          `json:"enabled"`
	VSVIPRef          string                        `json:"vsvip_ref"`
	EastWestPlacement bool                          `json:"east_west_placement"`
	DNSInfo           []interface{}                 `json:"dns_info"`
	VIP               []interface{}                 `json:"vip"`
}

type VirtualServiceConfigService struct {
	EnableHTTP2  bool `json:"enable_http2"`
	EnableSSL    bool `json:"enable_ssl"`
	Port         int  `json:"port"`
	PortRangeEnd int  `json:"port_range_end"`
}
