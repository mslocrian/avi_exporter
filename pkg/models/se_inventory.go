package models

type SeInventory struct {
	Count   int64 `json:"count"`
	Results []ServiceEngineInventory
}

type ServiceEngineInventory struct {
	Alert       ServiceEngineInventoryAlert       `json:"alert"`
	Config      ServiceEngineInventoryConfig      `json:"config"`
	HealthScore ServiceEngineInventoryHealthScore `json:"health_score"`
	Metrics     ServiceEngineInventoryMetrics     `json:"metrics"`
	RunTime     ServiceEngineInventoryRunTime     `json:"runtime"`
	UUID        string                            `json:"uuid"`
}

type ServiceEngineInventoryAlert struct {
	High   int64 `json:"high"`
	Low    int64 `json:"low"`
	Medium int64 `json:"medium"`
}

type ServiceEngineInventoryConfig struct {
	CloudRef      string `json:"cloud_ref"`
	EnableState   string `json:"enable_state"`
	MgmtIpAddress struct {
		Addr string `json:"addr"`
		Type string `json:"type"`
	} `json:"mgmt_ip_address"`
	Name               string   `json:"name"`
	SeGroupRef         string   `json:"se_group_ref"`
	TenantRef          string   `json:"tenant_ref"`
	URL                string   `json:"url"`
	UUID               string   `json:"uuid"`
	VirtualServiceRefs []string `json:"virtualservice_refs"`
	VSPerSERefs        []string `json:"vs_per_se_refs"`
}

type ServiceEngineInventoryHealthScore struct {
	AnomalyPenalty   float64 `json:"anomaly_penalty"`
	HealthScore      float64 `json:"health_score"`
	PerformanceScore float64 `json:"performance_score"`
	ResourcesPenalty float64 `json:"resources_penalty"`
	SecurityPenalty  float64 `json:"security_penalty"`
}

type ServiceEngineInventoryMetrics struct {
	SeIfAvgBandwdith struct {
		Timestamp string  `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"se_if.avg_bandwidth"`
	SeStatsAvgCpuUsage struct {
		Timestamp string  `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"se_stats.avg_cpu_usage"`
}

type ServiceEngineInventoryRunTime struct {
	AtCurrVer bool `json:"at_curr_ver"`
	GatewayUp bool `json:"gateway_up"`
	HbStatus  struct {
		LastHbReqSent    string `json:"last_hb_req_sent"`
		LastHbRespRecv   string `json:"last_hb_resp_recv"`
		NumHbMisses      int64  `json:"num_hb_misses"`
		NumOutstandingHb int64  `json:"num_outstanding_hb"`
	} `json:"hb_status"`
	InbandMgmt   bool   `json:"inband_mgmt"`
	MigrateState string `json:"migrate_state"`
	OnlineSince  string `json:"online_since"`
	OperStatus   struct {
		LastChangedTime struct {
			Secs  int64 `json:"secs"`
			Usecs int64 `json:"usecs"`
		} `json:"last_changed_time"`
		Reason     []string `json:"reason"`
		ReasonCode int64    `json:"reason_code"`
		State      string   `json:"state"`
	} `json:"oper_status"`
	PowerState         string `json:"power_state"`
	SeConnected        bool   `json:"se_connected"`
	SeGrpRebootPending bool   `json:"se_grp_reboot_pending"`
	SufficientMemory   bool   `json:"sufficient_memory"`
	Version            string `json:"version"`
	VinfraDiscovered   bool   `json:"vinfra_discovered"`
}
