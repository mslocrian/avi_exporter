package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

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

func fromJSONFile(path string, ob interface{}) (err error) {
	toReturn := ob
	openedFile, err := os.Open(path)
	defer openedFile.Close()
	if err != nil {
		// log.Infoln(err)
		return err
	}
	byteValue, err := ioutil.ReadAll(openedFile)
	if err != nil {
		// log.Infoln(err)
		return err
	}
	err = json.Unmarshal(byteValue, &toReturn)
	if err != nil {
		// log.Infoln(err)
		return err
	}
	return nil
}

func (o *Exporter) getDefaultMetrics(entityType string) (r DefaultMetrics, err error) {
	var path string
	r = DefaultMetrics{}
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

func (o *Exporter) setAllMetricsMap() (r GaugeOptsMap, err error) {
	r = make(GaugeOptsMap)
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
		r[v.Metric] = GaugeOpts{CustomLabels: []string{"name", "fqdn", "ipaddress", "pool", "tenant_uuid", "units", "controller"}, Type: "virtualservice", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	for _, v := range seDefaultMetrics {
		fName := strings.ReplaceAll(v.Metric, ".", "_")
		r[v.Metric] = GaugeOpts{CustomLabels: []string{"name", "entity_uuid", "fqdn", "ipaddress", "tenant_uuid", "units", "controller"}, Type: "serviceengine", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	for _, v := range controllerDefaultMetrics {
		fName := strings.ReplaceAll(v.Metric, ".", "_")
		r[v.Metric] = GaugeOpts{CustomLabels: []string{"name", "entity_uuid", "fqdn", "ipaddress", "tenant_uuid", "units", "controller"}, Type: "controller", GaugeOpts: prometheus.GaugeOpts{Name: fName, Help: v.Help}}
	}
	//////////////////////////////////////////////////////////////////////////////
	return r, nil
}

func (o *Exporter) setPromMetricsMap() (r GaugeOptsMap) {
	r = make(GaugeOptsMap)
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
	// r.connectionOpts = r.setConnectionOpts(username, password)
	r.connectionOpts = r.setConnectionOpts()
	r.GaugeOptsMap = r.setPromMetricsMap()
	r.logger = logger
	return
}

// func (o *Exporter) setConnectionOpts(username, password string) (r connectionOpts) {
func (o *Exporter) setConnectionOpts() (r connectionOpts) {
	r.username = os.Getenv("AVI_USERNAME")
	r.password = os.Getenv("AVI_PASSWORD")
	//r.username = username
	//r.password = username
	return
}

func (o *Exporter) setController(controller string) {
	o.connectionOpts.controller = controller
}

// connect establishes the avi connection.
func (o *Exporter) connect(cluster, tenant, api_version string) (r *clients.AviClient, err error) {
	// o.setConnectionOpts()
	o.setController(cluster)
	// simplify avi connection
	r, err = clients.NewAviClient(cluster, o.connectionOpts.username,
		session.SetPassword(o.connectionOpts.password),
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

func (o *Exporter) getVirtualServices() (r map[string]virtualServiceDef, err error) {
	vs, err := o.AviClient.VirtualService.GetAll()
	var pooluuid string

	if err != nil {
		return r, err
	}
	r = make(map[string]virtualServiceDef)
	for _, v := range vs {
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

		r[*v.UUID] = virtualServiceDef{Name: *v.Name, IPAddress: address, FQDN: strings.Join(dns, ","), PoolUUID: pooluuid}
	}
	return r, nil
}

func (o *Exporter) getClusterRuntime() (r map[string]clusterDef, err error) {
	resp := new(cluster)
	err = o.AviClient.AviSession.Get("/api/cluster", &resp)

	if err != nil {
		return r, err
	}
	r = make(map[string]clusterDef)
	for _, v := range resp.Nodes {
		address := v.IP.Addr
		dns, _ := net.LookupAddr(address)
		r[v.VMUUID] = clusterDef{Name: v.Name, IPAddress: address, FQDN: strings.Join(dns, ",")}
	}
	return r, nil
}

func (o *Exporter) getServiceEngines() (r map[string]seDef, err error) {
	se, err := o.AviClient.ServiceEngine.GetAll()
	if err != nil {
		return r, err
	}
	r = make(map[string]seDef)
	for _, v := range se {
		address := *v.MgmtVnic.VnicNetworks[0].IP.IPAddr.Addr
		dns, _ := net.LookupAddr(address)
		for k, v := range dns {
			dns[k] = strings.TrimSuffix(v, ".")
		}
		r[*v.UUID] = seDef{Name: *v.Name, IPAddress: address, FQDN: strings.Join(dns, ",")}
	}
	return r, nil
}

func (o *Exporter) getPools() (r map[string]poolDef, err error) {
	vs, err := o.AviClient.Pool.GetAll()
	r = make(map[string]poolDef)

	if err != nil {
		return r, err
	}

	for _, v := range vs {
		r[*v.UUID] = poolDef{Name: *v.Name}
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
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Connect to the cluster.
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	o.AviClient, err = o.connect(controller, tenant, api_version)
	if err != nil {
		return metrics, err
	}
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Set promMetrics.
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	err = o.setVirtualServiceMetrics()
	if err != nil {
		return metrics, err
	}
	err = o.setServiceEngineMetrics()
	if err != nil {
		return metrics, err
	}
	err = o.setControllerMetrics()
	if err != nil {
		return metrics, err
	}
	return o.metrics, err
}

func (o *Exporter) getVirtualServiceMetrics() (r [][]CollectionResponse, err error) {
	req := Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "virtualservice" {
			reqMetric := MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "VSERVER_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]CollectionResponse)
	err = o.AviClient.AviSession.Post("/api/analytics/metrics/collection", req, &resp)

	if err != nil {
		return r, err
	}

	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, nil
}

func (o *Exporter) getServiceEngineMetrics() (r [][]CollectionResponse, err error) {
	req := Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "serviceengine" {
			reqMetric := MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "SE_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]CollectionResponse)
	err = o.AviClient.AviSession.Post("/api/analytics/metrics/collection", req, &resp)
	if err != nil {
		return r, err
	}
	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, err
}

func (o *Exporter) getControllerMetrics() (r [][]CollectionResponse, err error) {
	req := Metrics{}
	for k, v := range o.GaugeOptsMap {
		if v.Type == "controller" {
			reqMetric := MetricRequest{}
			reqMetric.EntityUUID = "*"
			reqMetric.MetricEntity = "CONTROLLER_METRICS_ENTITY"
			reqMetric.Limit = 1
			reqMetric.MetricID = k
			reqMetric.Step = 5
			req.MetricRequests = append(req.MetricRequests, reqMetric)
		}
	}

	resp := make(map[string]map[string][]CollectionResponse)
	err = o.AviClient.AviSession.Post("/api/analytics/metrics/collection", req, &resp)
	if err != nil {
		return r, err
	}
	for _, s := range resp["series"] {
		r = append(r, s)
	}

	return r, nil
}

func (o *Exporter) setVirtualServiceMetrics() (err error) {
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Get lb objects for mapping.
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	vs, _ := o.getVirtualServices()
	pools, _ := o.getPools()
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	results, err := o.getVirtualServiceMetrics()
	if err != nil {
		return err
	}
	for _, v := range results {
		for _, v1 := range v {
			var labelnames = []string{"name", "pool", "tenant_uuid", "controller", "units", "fqdn", "ipaddress"}
			var labelvalues = []string{vs[v1.Header.EntityUUID].Name, pools[vs[v1.Header.EntityUUID].PoolUUID].Name, v1.Header.TenantUUID, o.connectionOpts.controller, v1.Header.Units, vs[v1.Header.EntityUUID].FQDN, vs[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_"+strings.Replace(v1.Header.Name, ".", "_", -1), "Service Engine Metrics", labelnames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelvalues...)
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
			var labelnames = []string{"tenant_uuid", "entity_uuid", "controller", "units", "name", "fqdn", "ipaddress"}
			var labelvalues = []string{v1.Header.TenantUUID, v1.Header.EntityUUID, o.connectionOpts.controller, v1.Header.Units, ses[v1.Header.EntityUUID].Name, ses[v1.Header.EntityUUID].FQDN, ses[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_"+strings.Replace(v1.Header.Name, ".", "_", -1), "Service Engine Metrics", labelnames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelvalues...)
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
			var labelnames = []string{"tenant_uuid", "entity_uuid", "controller", "units", "name", "fqdn", "ipaddress"}
			var labelvalues = []string{v1.Header.TenantUUID, v1.Header.EntityUUID, o.connectionOpts.controller, v1.Header.Units, runtime[v1.Header.EntityUUID].Name, runtime[v1.Header.EntityUUID].FQDN, runtime[v1.Header.EntityUUID].IPAddress}
			newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc("avi_"+strings.Replace(v1.Header.Name, ".", "_", -1), "Controller Metrics", labelnames, nil),
				prometheus.GaugeValue, v1.Data[len(v1.Data)-1].Value, labelvalues...)
			if err != nil {
				return err
			}
			o.metrics = append(o.metrics, newMetric)
		}
	}
	return nil
}
