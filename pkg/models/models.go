package models

import (
	"time"

	//"github.com/avinetworks/sdk/go/clients"
	//"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Gauges map[string]*prometheus.GaugeVec

// Connection describes the connection.
type Connection struct {
	UserName string
	Password string
	Host     string
	Tenant   string
}

// Runtime object.
type Runtime struct {
	NodeInfo struct {
		UUID        string `json:"uuid"`
		Version     string `json:"version"`
		MgmtIP      string `json:"mgmt_ip"`
		ClusterUUID string `json:"cluster_uuid"`
	} `json:"node_info"`
	NodeStates []struct {
		MgmtIP string `json:"mgmt_ip"`
		Role   string `json:"role"`
	} `json:"node_states"`
}

// ClusterResponse describes the response from the controller.
type ClusterResponse struct {
	NodeInfo struct {
		MgmtIP string `json:"mgmt_ip"`
	} `json:"node_info"`
}

// CollectionResponse describes the response sent from Avi's metrics/collection endpoint.
type CollectionResponse struct {
	Header struct {
		Statistics struct {
			Min        float64   `json:"min"`
			Trend      float64   `json:"trend"`
			Max        float64   `json:"max"`
			MaxTs      time.Time `json:"max_ts"`
			MinTs      time.Time `json:"min_ts"`
			NumSamples int       `json:"num_samples"`
			Mean       float64   `json:"mean"`
		} `json:"statistics"`
		MetricsMinScale      float64 `json:"metrics_min_scale"`
		MetricDescription    string  `json:"metric_description"`
		MetricsSumAggInvalid bool    `json:"metrics_sum_agg_invalid"`
		TenantUUID           string  `json:"tenant_uuid"`
		Priority             bool    `json:"priority"`
		EntityUUID           string  `json:"entity_uuid"`
		Units                string  `json:"units"`
		ObjIDType            string  `json:"obj_id_type"`
		DerivationData       struct {
			DerivationFn          string `json:"derivation_fn"`
			SecondOrderDerivation bool   `json:"second_order_derivation"`
			MetricIds             string `json:"metric_ids"`
		} `json:"derivation_data"`
		Name string `json:"name"`
	} `json:"header"`
	Data []struct {
		Timestamp time.Time `json:"timestamp"`
		Value     float64   `json:"value"`
	} `json:"data"`
}

// MetricList is the marshalled return payload of default metrics on Avi.
type MetricList struct {
	MetricsData map[string]struct {
		EntityTypes []string `json:"entity_types"`
		MetricUnits string   `json:"metric_units"`
		Description string   `json:"description"`
	} `json:"metrics_data"`
}

// ConnectionOpts describes the avi connection options.
type ConnectionOpts struct {
	Username   string
	Password   string
	Tenant     string
	Controller string
	ApiVersion string
}

// DefaultMetrics describes the default list of Avi metrics.
type DefaultMetrics []struct {
	Metric string `json:"metric"`
	Help   string `json:"help"`
}

// Gauge describes the prometheus gauge.
type Gauge struct {
	Name   string
	Entity string
	Units  string
	Value  float64
	Tenant string
	Leader string
}

// GaugeOptsMap lists all the GaugeOpts that will be registered.
type GaugeOptsMap map[string]GaugeOpts

// GaugeOpts describes the custom GaugeOpts definition for mapping.
type GaugeOpts struct {
	Type         string
	GaugeOpts    prometheus.GaugeOpts
	CustomLabels []string
}

// Metrics contains all the metrics.
type Metrics struct {
	MetricRequests []MetricRequest `json:"metric_requests"`
}

// MetricRequest describes the metric.
type MetricRequest struct {
	Step         int    `json:"step"`
	Limit        int    `json:"limit"`
	EntityUUID   string `json:"entity_uuid"`
	MetricEntity string `json:"metric_entity,omitempty"`
	MetricID     string `json:"metric_id"`
}

type VirtualServiceDef struct {
	Name      string
	PoolUUID  string
	IPAddress string `json:"ipaddress"`
	FQDN      string `json:"fqdn"`
}

type ClusterDef struct {
	IPAddress string `json:"ipaddress"`
	FQDN      string `json:"fqdn"`
	Name      string `json:"name"`
}

type SeDef struct {
	IPAddress string `json:"ipaddress"`
	FQDN      string `json:"fqdn"`
	Name      string `json:"name"`
}

type PoolDef struct {
	Name string
}

type Cluster struct {
	VirtualIP struct {
		Type string `json:"type"`
		Addr string `json:"addr"`
	} `json:"virtual_ip"`
	Nodes []struct {
		IP struct {
			Type string `json:"type"`
			Addr string `json:"addr"`
		} `json:"ip"`
		VMHostname string `json:"vm_hostname"`
		VMUUID     string `json:"vm_uuid"`
		Name       string `json:"name"`
		VMMor      string `json:"vm_mor"`
	} `json:"nodes"`
	TenantUUID string `json:"tenant_uuid"`
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
}

type BaseLicense struct {
	BurstCores   int           `json:"burst_cores"`
	Cores        int           `json:"cores"`
	CustomerName string        `json:"customer_name"`
	LicenseID    string        `json:"license_id"`
	LicenseTier  []string      `json:"license_tier"`
	LicenseTiers []LicenseTier `json:"license_tiers"`
	Licenses     []License     `json:"licenses"`
	MaxSes       int           `json:"max_ses"`
	Name         string        `json:"name"`
	Sockets      int           `json:"sockets"`
	//StartOn      time.Time     `json:"start_on"`
	StartOn string `json:"start_on"`
	UUID    string `json:"uuid"`
	//ValidUntil   time.Time     `json:"valid_until"`
	ValidUntil string `json:"valid_until"`
}

type LicenseTier struct {
	BurstCores int    `json:"burst_cores"`
	Cores      int    `json:"cores"`
	MaxSes     int    `json:"max_ses"`
	Sockets    int    `json:"soockets"`
	TierType   string `json:"tier_type"`
}

type License struct {
	//CreatedOn     time.Time `json:"created_on"`
	CreatedOn    string `json:"created_on"`
	CustomerName string `json:"customer_name"`
	//LicenseId     time.Time `json:"license_id"`
	LicenseId     string `json:"license_id"`
	LicenseName   string `json:"license_name"`
	LicenseString string `json:"license_string"`
	LicenseType   string `json:"license_type"`
	Sockets       int    `json:"sockets,omitempty"`
	Cores         int    `json:"cores,omitempty"`
	//StartOn       time.Time `json:"start_on"`
	StartOn  string `json:"start_on"`
	TierType string `json:"tier_type"`
	//ValidUntil    time.Time `json:"valid_until"`
	ValidUntil string `json:"valid_until"`
	Version    string `json:"version"`
}

type ServiceEngines struct {
	Count   int             `json:"count"`
	Results []ServiceEngine `json:"results"`
}

type ServiceEngine struct {
	LastModified       string                 `json:"_last_modified"`
	CloudRef           string                 `json:"cloud_ref"`
	ContainerMode      bool                   `json:"container_mode"`
	ContainerType      string                 `json:"controller_type"`
	ControllerCreated  bool                   `json:"controller_created"`
	ControllerIp       string                 `json:"controller_ip"`
	DataVnics          []interface{}          `json:"data_vnics"`
	EnableState        string                 `json:"enable_state"`
	Flavor             string                 `json:"flavor"`
	Hypervisor         string                 `json:"hypervisor"`
	InbandMgmt         bool                   `json:"inband_mgmt"`
	LicenseState       string                 `json:"license_state"`
	MgmtVnic           interface{}            `json:"mgmt_vnic"`
	MigrateState       string                 `json:"migrate_state"`
	Name               string                 `json:"name"`
	OnlineSince        string                 `json:"online_since"`
	Resources          ServiceEngineResources `json:"resources"`
	SeConnected        bool                   `json:"se_connected"`
	SeGroupRef         string                 `json:"se_group_ref"`
	SeGrpRebootPending bool                   `json:"se_grp_reboot_pending"`
	TenantRef          string                 `json:"tenant_ref"`
	URL                string                 `json:"url"`
	UUID               string                 `json:"uuid"`
	Version            string                 `json:"version"`
	VinfraDiscovered   bool                   `json:"vinfra_discovered"`
	VsRefs             []interface{}          `json:"vs_refs"`
}

type ServiceEngineResources struct {
	CoresPerSocket int  `json:"cores_per_socket"`
	Disk           int  `json:"disk"`
	HyperThreading bool `json:"hyper_threading"`
	Memory         int  `json:"memory"`
	NumVcpus       int  `json:"num_vpcus"`
	Sockets        int  `json:"sockets"`
}

type ServiceEngineMemDist struct {
	AppLearningMemoryMB int    `json:"app_learning_memory_mb"`
	Clusters            int    `json:"clusters"`
	ConfigMemoryMB      int    `json:"config_memory_mb"`
	ConnMemoryMB        int    `json:"conn_memory_mb"`
	ConnMemoryMBPerCore int    `json:"conn_memory_mb_per_core"`
	HugePages           int    `json:"huge_pages"`
	HypervisorType      int    `json:"hypervisor_type"`
	NumQueues           int    `json:"num_queues"`
	NumRXd              int    `json:"num_rxd"`
	NumTXd              int    `json:"num_txd"`
	OSReservedMemoryMB  int    `json:"os_reserved_memory_mb"`
	ProcID              string `json:"proc_id"`
	SeUUID              string `json:"se_uuid"`
	ShmConfigMemoryMB   int    `json:"shm_config_memory_mb"`
	ShmConnMemoryMB     int    `json:"shm_conn_memory_mb"`
	ShmMemoryMB         int    `json:"shm_memory_mb"`
}

type ServiceEngineSHMallocStats struct {
	SeUUID            string                           `json:"se_uuid"`
	ShMallocStatEntry []ServiceEngineSHMallocStatEntry `json:"sh_mallocstat_entry"`
}
type ServiceEngineSHMallocStatEntry struct {
	ShMallocTypeCnt  int    `json:"sh_malloc_type_cnt"`
	ShMallocTypeFail int    `json:"sh_malloc_type_fail"`
	ShMallocTypeName string `json:"sh_malloc_type_name"`
	ShMallocTypeSize int    `json:"sh_malloc_type_size"`
}
