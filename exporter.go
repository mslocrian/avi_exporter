package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mslocrian/avi_exporter/pkg/models"

	"github.com/avinetworks/sdk/go/clients"
	"github.com/avinetworks/sdk/go/session"
	"github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/common/log"
	"github.com/tidwall/pretty"
)

func formatAviRef(in string) string {
	uriArr := strings.SplitAfter(in, "/")
	return uriArr[len(uriArr)-1]
}

// Exporter describes the prometheus exporter.
type Exporter struct {
	GaugeOptsMap     models.GaugeOptsMap
	AviClient        *clients.AviClient
	connectionOpts   models.ConnectionOpts
	userMetricString string
	gauges           models.Gauges
	logger           log.Logger
	ses              *models.SeInventory
	vsinv            []models.VirtualServiceInventory
	metrics          []prometheus.Metric
	tenants          *models.TenantInventory
	clouds           *models.CloudInventory
	segroups         *models.SeGroupInventory
}

func fromJSONFile(path string, ob interface{}) (err error) {
	toReturn := ob
	openedFile, err := os.Open(path)
	defer openedFile.Close()
	if err != nil {
		return err
	}
	byteValue, err := ioutil.ReadAll(openedFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &toReturn)
	if err != nil {
		return err
	}
	return nil
}

func (o Exporter) getDefaultMetrics(entityType string) (r models.DefaultMetrics, err error) {
	var path string
	r = models.DefaultMetrics{}
	switch entityType {
	case "virtualservice":
		path = "lib/virtualservice_metrics.json"
	case "serviceengine":
		path = "lib/serviceengine_metrics.json"
	case "controller":
		path = "lib/controller_metrics.json"
	default:
		err = errors.New("entity type must be either: virtualserver, servicengine or controller")
		return r, err
	}

	err = fromJSONFile(path, &r)
	if err != nil {
		return r, err
	}
	return r, nil
}

func (o *Exporter) setAllMetricsMap() (r models.GaugeOptsMap, err error) {
	r = make(models.GaugeOptsMap)
	//////////////////////////////////////////////////////////////////////////////
	// Get default metrics.
	//////////////////////////////////////////////////////////////////////////////
	vsDefaultMetrics, err := o.getDefaultMetrics("virtualservice")
	if err != nil {
		return r, err
	}
	seDefaultMetrics, err := o.getDefaultMetrics("serviceengine")
	if err != nil {
		return r, err
	}
	controllerDefaultMetrics, err := o.getDefaultMetrics("controller")
	if err != nil {
		return r, err
	}
	//////////////////////////////////////////////////////////////////////////////
	// Populating default metrics. Leaving these as separate functions
	// in the event we want different GaugeOpts in the future.
	//////////////////////////////////////////////////////////////////////////////
	for _, v := range vsDefaultMetrics {
		fName := strings.ReplaceAll(v.Metric, ".", "_")
		r[v.Metric] = models.GaugeOpts{CustomLabels: []string{"name", "fqdn", "ipaddress", "pool", "tenant_uuid", "tenant", "units", "controller"}, Type: "virtualservice", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	for _, v := range seDefaultMetrics {
		fName := strings.ReplaceAll(v.Metric, ".", "_")
		r[v.Metric] = models.GaugeOpts{CustomLabels: []string{"name", "entity_uuid", "fqdn", "ipaddress", "tenant_uuid", "tenant", "units", "controller"}, Type: "serviceengine", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	for _, v := range controllerDefaultMetrics {
		fName := strings.ReplaceAll(v.Metric, ".", "_")
		r[v.Metric] = models.GaugeOpts{CustomLabels: []string{"name", "entity_uuid", "fqdn", "ipaddress", "tenant_uuid", "tenant", "units", "controller"}, Type: "controller", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	//////////////////////////////////////////////////////////////////////////////
	return r, nil
}

func (o *Exporter) setPromMetricsMap() (r models.GaugeOptsMap) {
	r = make(models.GaugeOptsMap)
	all, _ := o.setAllMetricsMap()
	if o.userMetricString == "" {
		r = all
		return
	}
	/////////////////////////////////////////////////////////
	// User provided metrics list
	/////////////////////////////////////////////////////////
	metrics := strings.Split(o.userMetricString, ",")
	for _, v := range metrics {
		r[v] = all[v]
	}
	return
}
func (o *Exporter) setUserMetrics() (r string) {
	r = os.Getenv("AVI_METRICS")
	return
}

// NewExporter constructor.
func NewExporter(username, password string, logger log.Logger) (r *Exporter) {
	r = new(Exporter)
	r.userMetricString = r.setUserMetrics()
	r.connectionOpts = r.setConnectionOpts()
	r.GaugeOptsMap = r.setPromMetricsMap()
	r.logger = logger
	return
}

// func (o *Exporter) setConnectionOpts(username, password string) (r connectionOpts) {
func (o *Exporter) setConnectionOpts() (r models.ConnectionOpts) {
	r.Username = os.Getenv("AVI_USERNAME")
	r.Password = os.Getenv("AVI_PASSWORD")
	return
}

func (o *Exporter) setController(controller string) {
	o.connectionOpts.Controller = controller
}

// connect establishes the avi connection.
func (o *Exporter) connect(cluster, tenant, api_version string) (r *clients.AviClient, err error) {
	o.setController(cluster)
	// simplify avi connection
	r, err = clients.NewAviClient(cluster, o.connectionOpts.Username,
		session.SetPassword(o.connectionOpts.Password),
		session.SetTenant(tenant),
		session.SetInsecure,
		session.SetVersion(api_version))
	return
}
func (o *Exporter) registerGauges() {
	o.gauges = make(map[string]*prometheus.GaugeVec)
	for k, v := range o.GaugeOptsMap {
		g := prometheus.NewGaugeVec(v.GaugeOpts, v.CustomLabels)
		o.gauges[k] = g
	}
}

// sortUniqueKeys sorts unique keys within a string array.
func sortUniqueKeys(in []string) ([]string, error) {
	var err error
	var resp []string
	respMap := make(map[string]string)
	for _, v := range in {
		respMap[v] = v
	}
	for _, v := range respMap {
		resp = append(resp, v)
	}
	sort.Strings(resp)
	return resp, err
}

func (o *Exporter) getVirtualServices() (r map[string]models.VirtualServiceDef, err error) {
	vs, err := o.AviClient.VirtualService.GetAll()
	var pooluuid string

	if err != nil {
		return r, err
	}
	r = make(map[string]models.VirtualServiceDef)
	for _, v := range vs {
		if v.Vip == nil {
			continue
		}
		vip := v.Vip[0]
		address := *vip.IPAddress.Addr
		dns, _ := net.LookupAddr(address)
		for k, v := range dns {
			dns[k] = strings.TrimSuffix(v, ".")
		}

		dns, err = sortUniqueKeys(dns)

		if v.PoolRef != nil {
			pooluuid = formatAviRef(*v.PoolRef)
		}

		r[*v.UUID] = models.VirtualServiceDef{Name: *v.Name, IPAddress: address, FQDN: strings.Join(dns, ","), PoolUUID: pooluuid}
	}
	return r, nil
}

func (o *Exporter) getClusterRuntime() (r map[string]models.ClusterDef, err error) {
	resp := new(models.Cluster)
	err = o.AviClient.AviSession.Get("/api/cluster", &resp)

	if err != nil {
		return r, err
	}
	r = make(map[string]models.ClusterDef)
	for _, v := range resp.Nodes {
		address := v.IP.Addr
		dns, _ := net.LookupAddr(address)
		r[v.VMUUID] = models.ClusterDef{Name: v.Name, IPAddress: address, FQDN: strings.Join(dns, ",")}
	}
	return r, nil
}

func (o *Exporter) getServiceEngines() (r map[string]models.SeDef, err error) {
	se, err := o.AviClient.ServiceEngine.GetAll()
	if err != nil {
		return r, err
	}
	r = make(map[string]models.SeDef)
	for _, v := range se {
		address := *v.MgmtVnic.VnicNetworks[0].IP.IPAddr.Addr
		dns, _ := net.LookupAddr(address)
		for k, v := range dns {
			dns[k] = strings.TrimSuffix(v, ".")
		}
		r[*v.UUID] = models.SeDef{Name: *v.Name, IPAddress: address, FQDN: strings.Join(dns, ",")}
	}
	return r, nil
}

func (o *Exporter) getPools() (r map[string]models.PoolDef, err error) {
	vs, err := o.AviClient.Pool.GetAll()
	r = make(map[string]models.PoolDef)

	if err != nil {
		return r, err
	}

	for _, v := range vs {
		r[*v.UUID] = models.PoolDef{Name: *v.Name}
	}
	return r, nil
}

// toPrettyJSON formats json output.
func toPrettyJSON(p interface{}) []byte {
	bytes, err := json.Marshal(p)
	if err != nil {
		// log.Infoln(err.Error())
	}
	return pretty.Pretty(bytes)
}

func CollectTarget(controller, username, password, tenant, api_version string, logger log.Logger) (metrics []prometheus.Metric, err error) {
	e := NewExporter(username, password, logger)
	e.registerGauges()
	metrics, err = e.Collect(controller, tenant, api_version)
	return metrics, err
}

// Collect retrieves metrics for Avi.
func (o *Exporter) Collect(controller, tenant, api_version string) (metrics []prometheus.Metric, err error) {
	/*
	 Connect to the cluster.
	*/
	o.AviClient, err = o.connect(controller, tenant, api_version)
	if err != nil {
		return metrics, err
	}
	err = o.AviClient.AviSession.Get("api/serviceengine-inventory?page_size=200", &o.ses)
	if err != nil {
		return metrics, err
	}

	err = o.AviClient.AviSession.Get("api/tenant?page_size=200", &o.tenants)
	if err != nil {
		return metrics, err
	}

	err = o.AviClient.AviSession.Get("api/cloud?page_size=200", &o.clouds)
	if err != nil {
		return metrics, err
	}

	err = o.AviClient.AviSession.Get("api/serviceenginegroup?page_size=200", &o.segroups)
	if err != nil {
		return metrics, err
	}

	// We need to pull un-exposed VS Faults (asymmetric vs's)
	page_iter := 1
	for {
		res := &models.VsInventory{}
		uri := fmt.Sprintf("api/virtualservice-inventory?page_size=200&page=%v", page_iter)
		err = o.AviClient.AviSession.Get(uri, res)
		count := res.Count
		for _, result := range res.Results {
			o.vsinv = append(o.vsinv, result)
		}
		if len(o.vsinv) >= int(count) {
			break
		}
		page_iter += 1
	}

	err = o.setVirtualServiceFaultMetrics()
	if err != nil {
		return metrics, err
	}

	/*
	 Set promMetrics.
	*/
	/*
		stegen - fix this
		" value:"" > label:<name:"pool" value:"" > label:<name:"tenant_uuid" value:"tenant-0a01a6d4-b3e0-4ffb-bd0d-73b30f2bf4b2" > label:<name:"units" value:"BITS_PER_SECOND" > gauge:<value:0 > } was collected before with the same name and label values
		* collected metric "avi_virtual_l4_server_avg_goodput" { label:<name:"controller" value:"lb-ctrl2-pub.or1.ne.adobe.net" > label:<name:"fqdn" value:"" > label:<name:"ipaddress" value:"" > label:<name:"name" value:"" > label:<name:"pool" value:"" > label:<name:"tenant_uuid" value:"tenant-0a01a6d4-b3e0-4ffb-bd0d-73b30f2bf4b2" > label:<name:"units" value:"BYTES_PER_SECOND" > gauge:<value:0 > } was collected before with the same name and label values

	*/

	/*
		err = o.setVirtualServiceMetrics()
		if err != nil {
			return metrics, err
		}
	*/

	err = o.setServiceEngineMetrics()
	if err != nil {
		return metrics, err
	}

	err = o.setControllerMetrics()
	if err != nil {
		return metrics, err
	}

	err = o.seMemDist()
	if err != nil {
		return metrics, err
	}

	err = o.seShMalloc()
	if err != nil {
		return metrics, err
	}

	// We may not have BGP Metrics on each controller
	err = o.seBgpPeerState()

	err = o.seVnicPortGroup()
	if err != nil {
		return metrics, err
	}

	err = o.seMissedHeartBeats()
	if err != nil {
		return metrics, err
	}

	err = o.getLicenseUsage()
	if err != nil {
		return metrics, err
	}

	err = o.getLicenseExpiration()
	if err != nil {
		return metrics, err
	}
	return o.metrics, err
}

func (o *Exporter) getVirtualServiceMetrics() (r [][]models.CollectionResponse, err error) {
	req := models.Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "virtualservice" {
			reqMetric := models.MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "VSERVER_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]models.CollectionResponse)
	err = o.AviClient.AviSession.Post("api/analytics/metrics/collection", req, &resp)

	if err != nil {
		return r, err
	}

	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, nil
}

func (o *Exporter) getServiceEngineMetrics() (r [][]models.CollectionResponse, err error) {
	req := models.Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "serviceengine" {
			reqMetric := models.MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "SE_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]models.CollectionResponse)
	err = o.AviClient.AviSession.Post("api/analytics/metrics/collection", req, &resp)
	if err != nil {
		return r, err
	}
	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, err
}

func (o *Exporter) getControllerMetrics() (r [][]models.CollectionResponse, err error) {
	req := models.Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "controller" {
			reqMetric := models.MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "CONTROLLER_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]models.CollectionResponse)
	err = o.AviClient.AviSession.Post("api/analytics/metrics/collection", req, &resp)
	if err != nil {
		return r, err
	}
	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, nil
}

func (o *Exporter) setVirtualServiceMetrics() (err error) {
	/*
	 Get lb objects for mapping.
	*/
	vs, _ := o.getVirtualServices()
	pools, _ := o.getPools()

	results, err := o.getVirtualServiceMetrics()
	if err != nil {
		return err
	}
	for _, v := range results {
		for _, v1 := range v {
			var labelNames = []string{"name", "pool", "tenant_uuid", "tenant", "controller", "units", "fqdn", "ipaddress"}
			var labelValues = []string{vs[v1.Header.EntityUUID].Name, pools[vs[v1.Header.EntityUUID].PoolUUID].Name, v1.Header.TenantUUID, o.getTenantNameFromUUID(v1.Header.TenantUUID), o.connectionOpts.Controller, v1.Header.Units, vs[v1.Header.EntityUUID].FQDN, vs[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_virtual_"+strings.Replace(v1.Header.Name, ".", "_", -1), "Virtual Service Metrics", labelNames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return nil
}

func (o *Exporter) setServiceEngineMetrics() (err error) {
	results, err := o.getServiceEngineMetrics()
	ses, _ := o.getServiceEngines()
	if err != nil {
		return err
	}
	for _, v := range results {
		for _, v1 := range v {
			var labelNames = []string{"tenant_uuid", "tenant", "entity_uuid", "controller", "units", "name", "fqdn", "ipaddress"}
			var labelValues = []string{v1.Header.TenantUUID, o.getTenantNameFromUUID(v1.Header.TenantUUID), v1.Header.EntityUUID, o.connectionOpts.Controller, v1.Header.Units, ses[v1.Header.EntityUUID].Name, ses[v1.Header.EntityUUID].FQDN, ses[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_"+strings.Replace(v1.Header.Name, ".", "_", -1), "Service Engine Metrics", labelNames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return nil
}

func (o *Exporter) setVirtualServiceFaultMetrics() (err error) {
	for _, vs := range o.vsinv {
		faults := vs.Faults.(map[string]interface{})
		if _, found := faults["shared_vip"]; found {
			tenantSplit := strings.Split(vs.Config.TenantRef, "/")
			cloudSplit := strings.Split(vs.Config.CloudRef, "/")
			seGroupSplit := strings.Split(vs.Config.SEGroupRef, "/")
			var labelNames = []string{"name", "uuid", "tenant", "cloud", "se_group", "controller"}
			var labelValues = []string{vs.Config.Name, vs.Config.UUID, o.getTenantNameFromUUID(tenantSplit[len(tenantSplit)-1]), o.getCloudNameFromUUID(cloudSplit[len(cloudSplit)-1]), o.getSEGroupNameFromUUID(seGroupSplit[len(seGroupSplit)-1]), o.connectionOpts.Controller}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_vs_fault_shared_vip", "Virtual Service Shared VIP Fault Metrics", labelNames, nil),
				prometheus.GaugeValue, 1, labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return nil
}

func (o *Exporter) setControllerMetrics() (err error) {
	results, err := o.getControllerMetrics()
	runtime, _ := o.getClusterRuntime()

	if err != nil {
		return err
	}
	for _, v := range results {
		for _, v1 := range v {
			var labelNames = []string{"tenant_uuid", "tenant", "entity_uuid", "controller", "units", "name", "fqdn", "ipaddress"}
			var labelValues = []string{v1.Header.TenantUUID, o.getTenantNameFromUUID(v1.Header.TenantUUID), v1.Header.EntityUUID, o.connectionOpts.Controller, v1.Header.Units, runtime[v1.Header.EntityUUID].Name, runtime[v1.Header.EntityUUID].FQDN, runtime[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_"+strings.ReplaceAll(v1.Header.Name, ".", "_"), "Controller Metrics", labelNames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return nil
}

func (o *Exporter) seMemDist() (err error) {
	for _, se := range o.ses.Results {
		var memDist []models.ServiceEngineMemDist
		err = o.AviClient.AviSession.Get("api/serviceengine/"+se.UUID+"/memdist", &memDist)
		if err != nil {
			e := err.(session.AviError)
			if e.HttpStatusCode == 500 {
				level.Error(o.logger).Log("msg", "There was an error collecting se memdist stats", "error", fmt.Sprintf("%#v", err))
				continue
			} else {
				return err
			}
		}

		for _, dist := range memDist {
			var labelNames = []string{"controller", "uuid", "ip", "proc_id"}
			var labelValues = []string{o.connectionOpts.Controller, se.UUID, se.Config.MgmtIpAddress.Addr, dist.ProcID}

			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_dist_shm_memory_mb", "AVI SE Memory Distribution Shared Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ShmMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_dist_app_learning_memory_mb", "AVI SE Memory Distribution App Learning Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.AppLearningMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_dist_clusters", "AVI SE Memory Distribution Clusters", labelNames, nil),
				prometheus.GaugeValue, float64(dist.Clusters), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_config_memory_mb", "AVI SE Memory Distribution Config Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ConfigMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_conn_memory_mb", "AVI SE Memory Distribution Connection Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ConnMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_conn_memory_per_core_mb", "AVI SE Memory Distribution Connection Memory Per Core MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ConnMemoryMBPerCore), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_huge_pages", "AVI SE Memory Distribution Huge Pages", labelNames, nil),
				prometheus.GaugeValue, float64(dist.HugePages), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_hypervisor_type", "AVI SE Memory Distribution Hypervisor Type", labelNames, nil),
				prometheus.GaugeValue, float64(dist.HugePages), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_num_queues", "AVI SE Memory Distribution Num Queues", labelNames, nil),
				prometheus.GaugeValue, float64(dist.NumQueues), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_num_recv", "AVI SE Memory Distribution Num Received", labelNames, nil),
				prometheus.GaugeValue, float64(dist.NumRXd), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_num_xmit", "AVI SE Memory Distribution Num Transmitted", labelNames, nil),
				prometheus.GaugeValue, float64(dist.NumTXd), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_num_os_reserved_memory_mb", "AVI SE Memory Distribution OS Reserved Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.OSReservedMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_shm_config_memory_mb", "AVI SE Memory Distribution Shared Config Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ShmConfigMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_mem_shm_conn_memory_mb", "AVI SE Memory Distribution Shared Connection Memory MB", labelNames, nil),
				prometheus.GaugeValue, float64(dist.ShmConnMemoryMB), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return err
}

func (o *Exporter) seShMalloc() (err error) {
	for _, se := range o.ses.Results {
		var shMalloc []models.ServiceEngineSHMallocStats
		err = o.AviClient.AviSession.Get("api/serviceengine/"+se.UUID+"/shmallocstats", &shMalloc)
		if err != nil {
			e := err.(session.AviError)
			if e.HttpStatusCode == 500 {
				level.Error(o.logger).Log("msg", "There was an error collecting se shmalloc stats", "error", fmt.Sprintf("%#v", err))
				continue
			} else {
				return err
			}
		}
		for _, outerStats := range shMalloc {
			for _, shMallocStat := range outerStats.ShMallocStatEntry {
				var labelNames = []string{"controller", "uuid", "ip"}
				var labelValues = []string{o.connectionOpts.Controller, se.UUID, se.Config.MgmtIpAddress.Addr}

				shMallocMetricName := strings.ToLower(shMallocStat.ShMallocTypeName)

				metricNameSize := fmt.Sprintf("avi_%s_size", shMallocMetricName)
				metricNameFail := fmt.Sprintf("avi_%s_fail", shMallocMetricName)
				metricNameCount := fmt.Sprintf("avi_%s_count", shMallocMetricName)

				newMetricSize, err := prometheus.NewConstMetric(prometheus.NewDesc(metricNameSize, "AVI SE Shared Malloc Size Entry", labelNames, nil),
					prometheus.GaugeValue, float64(shMallocStat.ShMallocTypeSize), labelValues...)
				if err != nil {
					return err
				}

				newMetricFail, err := prometheus.NewConstMetric(prometheus.NewDesc(metricNameFail, "AVI SE Shared Malloc Fail Entry", labelNames, nil),
					prometheus.GaugeValue, float64(shMallocStat.ShMallocTypeFail), labelValues...)
				if err != nil {
					return err
				}

				newMetricCount, err := prometheus.NewConstMetric(prometheus.NewDesc(metricNameCount, "AVI SE Shared Malloc Count Entry", labelNames, nil),
					prometheus.GaugeValue, float64(shMallocStat.ShMallocTypeCnt), labelValues...)
				if err != nil {
					return err
				}

				o.metrics = append(o.metrics, newMetricSize)
				o.metrics = append(o.metrics, newMetricFail)
				o.metrics = append(o.metrics, newMetricCount)
			}
		}
	}
	return err
}

func (o *Exporter) seBgpPeerState() (err error) {
	for _, se := range o.ses.Results {
		var seBGP []models.SeBGP
		err = o.AviClient.AviSession.Get("api/serviceengine/"+se.UUID+"/bgp", &seBGP)
		if err != nil {
			e := err.(session.AviError)
			if e.HttpStatusCode == 500 {
				level.Error(o.logger).Log("msg", "There was an error collecting se bgp stats", "error", fmt.Sprintf("%#v", err))
				continue
			} else {
				return err
			}
		}

		for _, peer := range seBGP {
			var labelNames = []string{"controller", "uuid", "ip", "vrf"}
			var labelValues = []string{o.connectionOpts.Controller, se.UUID, se.Config.MgmtIpAddress.Addr, peer.Name}

			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_se_bgp_peer_count", "AVI SE BGP Peer Count", labelNames, nil),
				prometheus.GaugeValue, float64(len(peer.Peers)), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_bgp_vs_count", "AVI SE BGP Peer VS Count", labelNames, nil),
				prometheus.GaugeValue, float64(len(peer.VSNames)), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_bgp_route_count", "AVI SE BGP Peer Route Count", labelNames, nil),
				prometheus.GaugeValue, float64(len(peer.Routes)), labelValues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)

			for _, p := range peer.Peers {
				var labelNames = []string{"controller", "uuid", "ip", "vrf", "peer_ip", "peer_state", "remote_as"}
				var labelValues = []string{o.connectionOpts.Controller, se.UUID, se.Config.MgmtIpAddress.Addr, peer.Name, p.PeerIP, p.PeerState, fmt.Sprintf("%v", p.RemoteAS)}

				newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_bgp_peer_state", "AVI SE BGP Peer State", labelNames, nil),
					prometheus.GaugeValue, float64(p.Active), labelValues...)
				if err != nil {
					return err
				}
				o.metrics = append(o.metrics, newMetric)
			}
		}
	}
	return err
}

func (o *Exporter) seVnicPortGroup() (err error) {
	return err
}

func (o *Exporter) seMissedHeartBeats() (err error) {
	for _, se := range o.ses.Results {
		// TODO(stegen) add the timestamps?
		var labelNames = []string{"controller", "uuid", "ip"}
		var labelValues = []string{o.connectionOpts.Controller, se.UUID, se.Config.MgmtIpAddress.Addr}

		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_se_missed_heartbeats", "AVI Number SE Heartbeat Misses", labelNames, nil),
			prometheus.GaugeValue, float64(se.RunTime.HbStatus.NumHbMisses), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)

		newMetric, err = prometheus.NewConstMetric(prometheus.NewDesc("avi_se_outstanding_heartbeats", "AVI Number SE Outstanding Heartbeat", labelNames, nil),
			prometheus.GaugeValue, float64(se.RunTime.HbStatus.NumOutstandingHb), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}
	return err
}

func (o *Exporter) getLicenseUsage() (err error) {
	var res interface{}
	err = o.AviClient.AviSession.Get("api/licenseusage?limit=365&step=86400", &res)
	if err != nil {
		return err
	}

	licensing := res.(map[string]interface{})

	var labelNames = []string{"controller"}
	var labelValues = []string{o.connectionOpts.Controller}

	if _, ok := licensing["licensed_ses"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_ses_total", "AVI Total Licensed Service Engines", labelNames, nil),
			prometheus.GaugeValue, licensing["licensed_ses"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["num_ses"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_ses_used", "AVI Total Used Service Engines", labelNames, nil),
			prometheus.GaugeValue, licensing["num_ses"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["licensed_cores"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_cores_total", "AVI Total Licensed Cores", labelNames, nil),
			prometheus.GaugeValue, licensing["licensed_cores"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["licensed_service_cores"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_service_cores_total", "AVI Total Licensed Cores", labelNames, nil),
			prometheus.GaugeValue, licensing["licensed_service_cores"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["consumed_service_cores"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_consumed_service_cores_total", "AVI Total Licensed Cores", labelNames, nil),
			prometheus.GaugeValue, licensing["consumed_service_cores"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["num_se_vcpus"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_cores_used", "AVI Total Used Cores", labelNames, nil),
			prometheus.GaugeValue, licensing["num_se_vcpus"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["licensed_sockets"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_sockets_total", "AVI Total Licensed Sockets", labelNames, nil),
			prometheus.GaugeValue, licensing["licensed_sockets"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	if _, ok := licensing["num_sockets"]; ok {
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_licensed_sockets_used", "AVI Total Used Sockets", labelNames, nil),
			prometheus.GaugeValue, licensing["num_sockets"].(float64), labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}

	return err
}

func (o *Exporter) getLicenseExpiration() (err error) {
	var licenses models.BaseLicense
	timeLayout := "2006-01-02T15:04:05"
	err = o.AviClient.AviSession.Get("api/license", &licenses)
	if err != nil {
		return err
	}

	for _, l := range licenses.Licenses {
		var labelNames = []string{"controller", "license_id"}
		var labelValues = []string{o.connectionOpts.Controller, l.LicenseId}
		validUntil, err := time.Parse(timeLayout, l.ValidUntil)
		if err != nil {
			return err
		}

		expires := validUntil.Sub(time.Now())
		newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_license_expiration_days", "AVI License Expiration", labelNames, nil),
			prometheus.GaugeValue, expires.Hours()/24, labelValues...)
		if err != nil {
			return err
		}
		o.metrics = append(o.metrics, newMetric)
	}
	return nil
}

func (o *Exporter) getTenantNameFromUUID(uuid string) string {
	for _, tenant := range o.tenants.Results {
		tenantMap := tenant.(map[string]interface{})
		if tenantMap["uuid"] == uuid {
			return tenantMap["name"].(string)
		}
	}
	return "unknown"
}

func (o *Exporter) getCloudNameFromUUID(uuid string) string {
	for _, cloud := range o.clouds.Results {
		cloudMap := cloud.(map[string]interface{})
		if cloudMap["uuid"] == uuid {
			return cloudMap["name"].(string)
		}
	}
	return "unknown"
}

func (o *Exporter) getSEGroupNameFromUUID(uuid string) string {
	for _, seg := range o.segroups.Results {
		segMap := seg.(map[string]interface{})
		if segMap["uuid"] == uuid {
			return segMap["name"].(string)
		}
	}
	return "unknown"
}
