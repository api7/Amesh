package apisix

type LoadBalanceType string

const (
	LoadBalanceTypeChash      LoadBalanceType = "chash"
	LoadBalanceTypeRoundrobin LoadBalanceType = "roundrobin"
	LoadBalanceTypeEwma       LoadBalanceType = "ewma"
	LoadBalanceTypeLeaseConn  LoadBalanceType = "lease_conn"
)

type HashOn string

const (
	HashOnVars            HashOn = "vars"
	HashOnHeader          HashOn = "header"
	HashOnCookie          HashOn = "cookie"
	HashOnConsumer        HashOn = "consumer"
	HashOnVarsCombination HashOn = "vars_combination"
)

type ProtocolScheme string

const (
	ProtocolSchemeGrpc  ProtocolScheme = "grpc"
	ProtocolSchemeGrpcs ProtocolScheme = "grpcs"
	ProtocolSchemeHttp  ProtocolScheme = "http"
	ProtocolSchemeHttps ProtocolScheme = "https"
	ProtocolSchemeTcp   ProtocolScheme = "tcp"
	ProtocolSchemeTls   ProtocolScheme = "tls"
	ProtocolSchemeUdp   ProtocolScheme = "udp"
)

type HostPassingStrategy string

const (
	HostPassingStrategyPass    HostPassingStrategy = "pass"
	HostPassingStrategyNode    HostPassingStrategy = "node"
	HostPassingStrategyRewrite HostPassingStrategy = "rewrite"
)

type Node struct {
	// The endpoint host (could be IPv4/IPv6 or domain).
	Host string `json:"host,omitempty"`
	// The endpoint port.
	Port int32 `json:"port,omitempty"`
	// The endpoint weight.
	Weight int32 `json:"weight,omitempty"`
	// The metadata of node
	Metadata map[string]interface{} `json:"metadata,omitempty" `
}

type ActiveHealthCheckHealthy struct {
	// The interval to send a probe request.
	Interval int32 `json:"interval,omitempty"`
	// Probes with status codes in this array will be treated as healthy.
	HttpStatuses []int32 `json:"http_statuses,omitempty"`
	// How many consecutive success times should meet before a node is set to healthy.
	Successes int32 `json:"successes,omitempty"`
}

type ActiveHealthCheckUnhealthy struct {
	// The interval to send a probe request.
	Interval int32 `json:"interval,omitempty"`
	// Probes with status codes in this array will be treated as unhealthy.
	HttpStatuses []int32 `json:"http_statuses,omitempty"`
	// How many consecutive failures (http) occur should meet before a node is set to healthy.
	HttpFailures int32 `json:"http_failures,omitempty"`
	// How many consecutive failures (tcp) occur should meet before a node is set to healthy.
	TcpFailures int32 `json:"tcp_failures,omitempty"`
	// How many consecutive timeouts occur should meet before a node is set to healthy.
	Timeouts int32 `json:"timeouts,omitempty"`
}

type HealthCheckType string

const (
	HealthCheckTypeHttp  HealthCheckType = "http"
	HealthCheckTypeHttps HealthCheckType = "https"
	HealthCheckTypeTcp   HealthCheckType = "tcp"
)

type ActiveHealthCheck struct {
	// The health check probe type.
	Type string `json:"type,omitempty"`
	// Timeout setting for the probe requests.
	Timeout float64 `json:"timeout,omitempty"`
	// How many probes can be sent simultaneously.
	Concurrency int32 `json:"concurrency,omitempty"`
	// Host value for HTTP probes.
	Host string `json:"host,omitempty"`
	// Specified port for the probe to sent.
	Port int32 `json:"port,omitempty"`
	// the URI path for HTTP probes.
	HttpPath string `json:"http_path,omitempty"`
	// Whether to verify the TLS/SSL certificate.
	HttpsVerifyCertificate bool `json:"https_verify_certificate,omitempty"`
	// health check for judging nodes become healthy.
	Healthy *ActiveHealthCheckHealthy `json:"healthy,omitempty"`
	// health check for judging nodes become unhealthy.
	Unhealthy *ActiveHealthCheckUnhealthy `json:"unhealthy,omitempty"`
	// The extra request headers to carry for HTTP probes.
	ReqHeaders []string `json:"req_headers,omitempty"`
}

type PassiveHealthCheckHealthy struct {
	// Probes with status codes in this array will be treated as healthy.
	HttpStatuses []int32 `json:"http_statuses,omitempty"`
	// How many consecutive success times should meet before a node is set to healthy.
	Successes int32 `json:"successes,omitempty"`
}

type PassiveHealthCheckUnhealthy struct {
	// Probes with status codes in this array will be treated as unhealthy.
	HttpStatuses []int32 `json:"http_statuses,omitempty"`
	// How many consecutive failures (http) occur should meet before a node is set to healthy.
	HttpFailures int32 `json:"http_failures,omitempty"`
	// How many consecutive failures (tcp) occur should meet before a node is set to healthy.
	TcpFailures int32 `json:"tcp_failures,omitempty"`
	// How many consecutive timeouts occur should meet before a node is set to healthy.
	Timeouts int32 `json:"timeouts,omitempty"`
}

type PassiveHealthCheck struct {
	// The health check probe type.
	Type      string                       `json:"type,omitempty"`
	Healthy   *PassiveHealthCheckHealthy   `json:"healthy,omitempty"`
	Unhealthy *PassiveHealthCheckUnhealthy `json:"unhealthy,omitempty"`
}

type HealthCheck struct {
	// Active health check settings.
	Active *ActiveHealthCheck `json:"active,omitempty"`
	// Passive health check settings.
	Passive *PassiveHealthCheck `json:"passive,omitempty"`
}

// UpstreamTLSConfig defines the certificate and private key used to communicate with
// the upstream.
type UpstreamTLSConfig struct {
	// client_cert is the client certificate.
	ClientCert string `json:"client_cert,omitempty"`
	// client_key is the private key of the client_cert.
	ClientKey string `json:"client_key,omitempty"`
}

type KeepalivePool struct {
	// size indicates how many connections can be put into the connection pool,
	// when the threshold is reached, extra connection will be closed.
	Size int32 `json:"size,omitempty"`
	// requests indicates how many requests (at most) can be processed in a connection,
	// when the threshold is reached, connection won't be reused and will be closed.
	Requests int32 `json:"requests,omitempty"`
	// idle_timeout controls how long a connection can be stayed in the connection pool,
	// when the threshold is reached, connection will be closed.
	IdleTimeout float64 `json:"idle_timeout,omitempty"`
}

// An Upstream is the abstraction of service/cluster. It contains
// settings about how to communicate with the service/cluster correctly.
type Upstream struct {
	// create_time indicate the create timestamp of this route.
	CreateTime int64 `json:"create_time,omitempty"`
	// update_time indicate the last update timestamp of this route.
	UpdateTime int64 `json:"update_time,omitempty"`
	// Upstream nodes.
	Nodes []*Node `json:"nodes,omitempty"`
	// How many times a request can be retried while communicating to the upstream,
	// note request can be retried only if no bytes are sent to client.
	Retries int32 `json:"retries,omitempty"`
	// Timeout settings for this upstream.
	Timeout *Timeout `json:"timeout,omitempty"`
	// tls contains the TLS settings used to communicate with this upstream.
	Tls *UpstreamTLSConfig `json:"tls,omitempty"`
	// KeepalivePool controls the connection pool settings for this upstream.
	KeepalivePool *KeepalivePool `json:"keepalive_pool,omitempty"`
	// The load balancing algorithm.
	Type LoadBalanceType `json:"type,omitempty"`
	// The health check settings for this upstream.
	Check *HealthCheck `json:"check,omitempty"`
	// The scope of hash key, this field is only in effective
	// if type is "chash".
	HashOn HashOn `json:"hash_on,omitempty"`
	// The hash key, this field is only in effective
	// if type is "chash".
	Key string `json:"key,omitempty"`
	// The communication protocol to use.
	Scheme ProtocolScheme `json:"scheme,omitempty"`
	// labels contains some labels for the sake of management.
	Labels map[string]string `json:"labels,omitempty" `
	// The host passing strategy.
	PassHost HostPassingStrategy `json:"pass_host,omitempty"`
	// The HTTP Host header to use when sending requests to this upstream.
	UpstreamHost string `json:"upstream_host,omitempty"`
	// The upstream name, it's useful for the logging but it's not required.
	Name string `json:"name,omitempty"`
	// Textual descriptions used to describe the upstream use.
	Desc string `json:"desc,omitempty"`
	// The upstream id.
	Id string `json:"id,omitempty"`
}
