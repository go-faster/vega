// Package flow implements ClickHouse table schema for cilium hubble flow events.
package flow

import (
	"fmt"
	"net/netip"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/api/v1/observer"
	"github.com/go-faster/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func NewDDL(tableName string) string {
	// DDL for ClickHouse table.
	const ddl = `
CREATE TABLE IF NOT EXISTS %s
(
    timestamp                 DateTime64(9),

    -- index for time-based queries
    INDEX timestamp_idx timestamp TYPE minmax GRANULARITY 1,

    flow_type Enum8(
        'UNKNOWN_TYPE' = 0,
        'L3_L4'        = 1,
        'L7'           = 2,
        'SOCK'         = 3
    ) DEFAULT 'UNKNOWN_TYPE',

    verdict Enum8(
      'VERDICT_UNKNOWN' = 0,
      'FORWARDED'       = 1,
      'DROPPED'         = 2,
      'ERROR'           = 3,
      'AUDIT'           = 4,
      'REDIRECTED'      = 5,
      'TRACED'          = 6,
      'TRANSLATED'      = 7
    ) DEFAULT 'VERDICT_UNKNOWN',
    -- only applicable to verdict = DROPPED.
    drop_reason LowCardinality(String),

    -- Name of the node where this event was observed.
    node_name                 LowCardinality(String),
    is_reply                  Nullable(Bool),

    src_names Array(String),
    dst_names Array(String),

    -- cilium event type
    event_type     Int32,
    event_sub_type Int32,

    endpoint_src_id           UInt32,
    endpoint_src_identity     UInt32,
    endpoint_src_namespace    LowCardinality(String),
    endpoint_src_pod_name     LowCardinality(String),
    -- labels
    endpoint_src_labels  Array(LowCardinality(String)),
    -- workloads
    endpoint_src_workloads_names Array(LowCardinality(String)),
    endpoint_src_workloads_kinds Array(LowCardinality(String)),

    -- Destination
    endpoint_dst_id           UInt32,
    endpoint_dst_identity     UInt32,
    endpoint_dst_namespace    LowCardinality(String),
    endpoint_dst_pod_name     LowCardinality(String),
    -- labels
    endpoint_dst_labels      Array(LowCardinality(String)),
    -- workloads
    endpoint_dst_workloads_names Array(LowCardinality(String)),
    endpoint_dst_workloads_kinds Array(LowCardinality(String)),

    direction Enum8(
    	'UNKNOWN' = 0,
    	'DIRECT'  = 1,
    	'INVERSE' = 2
    ) DEFAULT 'UNKNOWN',

    -- k8s materialized fields
    k8s_pod       LowCardinality(String),
    k8s_container LowCardinality(String),
    k8s_ns        LowCardinality(String),

    -- Peer information.
    -- k8s materialized fields
    k8s_peer_pod       LowCardinality(String),
    k8s_peer_container LowCardinality(String),
    k8s_peer_ns        LowCardinality(String),

    traffic_direction Enum8(
        'TRAFFIC_DIRECTION_UNKNOWN' = 0,
        'INGRESS' = 1,
        'EGRESS'  = 2
    ) DEFAULT 'TRAFFIC_DIRECTION_UNKNOWN',

    policy_match_type UInt32,

    trace_observation_point Enum8(
        'UNKNOWN_POINT' = 0,
        -- TO_PROXY indicates network packets are transmitted towards the l7 proxy.
        'TO_PROXY' = 1,
        -- TO_HOST indicates network packets are transmitted towards the host
        -- namespace.
        'TO_HOST' = 2,
        -- TO_STACK indicates network packets are transmitted towards the Linux
        -- kernel network stack on host machine.
        'TO_STACK' = 3,
        -- TO_OVERLAY indicates network packets are transmitted towards the tunnel
        -- device.
        'TO_OVERLAY' = 4,
        -- TO_ENDPOINT indicates network packets are transmitted towards endpoints
        -- (containers).
        'TO_ENDPOINT' = 101,
        -- FROM_ENDPOINT indicates network packets were received from endpoints
        -- (containers).
        'FROM_ENDPOINT' = 5,
        -- FROM_PROXY indicates network packets were received from the l7 proxy.
        'FROM_PROXY' = 6,
        -- FROM_HOST indicates network packets were received from the host
        -- namespace.
        'FROM_HOST' = 7,
        -- FROM_STACK indicates network packets were received from the Linux kernel
        -- network stack on host machine.
        'FROM_STACK' = 8,
        -- FROM_OVERLAY indicates network packets were received from the tunnel
        -- device.
        'FROM_OVERLAY' = 9,
        -- FROM_NETWORK indicates network packets were received from native
        -- devices.
        'FROM_NETWORK' = 10,
        -- TO_NETWORK indicates network packets are transmitted towards native
        -- devices.
        'TO_NETWORK' = 11
    ) DEFAULT 'UNKNOWN_POINT',

    interface_index UInt32,
    interface_name  LowCardinality(String),

    proxy_port UInt32,

    trace_id String, -- trace context

    sock_xlate_point Enum8(
        'SOCK_XLATE_POINT_UNKNOWN'            = 0,
		'SOCK_XLATE_POINT_PRE_DIRECTION_FWD'  = 1,
		'SOCK_XLATE_POINT_POST_DIRECTION_FWD' = 2,
		'SOCK_XLATE_POINT_PRE_DIRECTION_REV'  = 3,
		'SOCK_XLATE_POINT_POST_DIRECTION_REV' = 4
	) DEFAULT 'SOCK_XLATE_POINT_UNKNOWN',

    socket_cookie UInt64,

    cgroup_id UInt64,


    -- L2 fields
    -- FIXME: use fixed string?
    ethernet_src  LowCardinality(String),
    ethernet_dst  LowCardinality(String),

    -- L3 fields
    ipv4_src  IPv4,
    ipv4_dst  IPv4,
    ipv6_src  IPv6,
    ipv6_dst  IPv6,
    ip_version  Enum8(
        'UNKNOWN' = 0,
        'IPv4'    = 4,
        'IPv6'    = 6
    ) default 'UNKNOWN',
    -- https://github.com/cilium/cilium/blob/ba0ed147bd5bb342f67b1794c2ad13c6e99d5236/pkg/monitor/datapath_trace.go#L27
    ip_encrypted  Bool,

    -- L4 fields
    l4_protocol Enum8(
        'UNKNOWN' = 0,
        'TCP'     = 1,
        'UDP'     = 2,
        'ICMPv4'  = 3,
        'ICMPv6'  = 4,
        'SCTP'    = 5
    ),

    -- Only for TCP, UDP and SCTP
    l4_src_port   UInt32,
    l4_dst_port   UInt32,

    -- Only for TCP
    l4_tcp_flags Array(Enum8(
        'FIN' = 1,
        'SYN' = 2,
        'RST' = 3,
        'PSH' = 4,
        'ACK' = 5,
        'URG' = 6,
        'ECE' = 7,
        'CWR' = 8,
        'NS'  = 9
    )),

    -- Only for ICMP(v4, v6)
    l4_icmp_type  UInt32,
    l4_icmp_code  UInt32,

    -- L7 fields
    l7_flow_type Enum8(
        'UNKNOWN_L7_TYPE'  = 0,
        'REQUEST'  = 1,
        'RESPONSE' = 2,
        'SAMPLE'   = 3
    ) default 'UNKNOWN_L7_TYPE',

    l7_protocol Enum8(
        'UNKNOWN' = 0,
        'DNS'     = 1,
        'HTTP'    = 2,
        'Kafka'   = 3
    ) default 'UNKNOWN',

    l7_latency_ns             UInt64,

    -- DNS fields
    l7_dns_query              String,
    l7_dns_ttl                UInt32,
    l7_dns_response_code      UInt16,
    l7_dns_response_ips       Array(String),
    l7_dns_response_cnames    Array(String),
    -- List of question types
    l7_dns_qtypes Array(String),
    -- List of answer types
    l7_dns_rrtypes Array(String),
    -- Corresponds to DNSDataSource defined in:
    --   https://github.com/cilium/cilium/blob/04f3889d627774f79e56d14ddbc165b3169e2d01/pkg/proxy/accesslog/record.go#L253
    l7_dns_observation_source String,

    -- HTTP fields
    l7_http_code              UInt16,
    l7_http_method            LowCardinality(String),
    l7_http_url               String,
    l7_http_protocol          LowCardinality(String),
    l7_http_headers_keys      Array(LowCardinality(String)),
    l7_http_headers_values    Array(String),

    -- Kafka fields
    l7_kafka_error_code       UInt32,
    l7_kafka_api_version      UInt32,
    l7_kafka_api_key          String,
    l7_kafka_correlation_id   Int32,
    l7_kafka_topic            String
)
    ENGINE = MergeTree()
        PARTITION BY toYearWeek(timestamp)
        ORDER BY (k8s_container, k8s_pod, timestamp)
`
	return fmt.Sprintf(ddl, tableName)
}

// DDL for ClickHouse table.
var DDL = NewDDL("flows")

// Table is wrapper for ClickHouse columns that simplifies data ingestion.
type Table struct {
	name       string
	flowType   proto.ColEnum
	verdict    proto.ColEnum
	dropReason proto.ColLowCardinality[string]
	nodeName   proto.ColLowCardinality[string]
	isReply    proto.ColNullable[bool]

	srcNames proto.ColArr[string]
	dstNames proto.ColArr[string]

	eventType    proto.ColInt32
	eventSubType proto.ColInt32

	endpointSrcID             proto.ColUInt32
	endpointSrcIdentity       proto.ColUInt32
	endpointSrcNamespace      proto.ColLowCardinality[string]
	endpointSrcPodName        proto.ColLowCardinality[string]
	endpointSrcLabels         proto.ColArr[string]
	endpointSrcWorkloadsNames proto.ColArr[string]
	endpointSrcWorkloadsKinds proto.ColArr[string]

	endpointDstID             proto.ColUInt32
	endpointDstIdentity       proto.ColUInt32
	endpointDstNamespace      proto.ColLowCardinality[string]
	endpointDstPodName        proto.ColLowCardinality[string]
	endpointDstLabels         proto.ColArr[string]
	endpointDstWorkloadsNames proto.ColArr[string]
	endpointDstWorkloadsKinds proto.ColArr[string]

	trafficDirection      proto.ColEnum
	policyMatchType       proto.ColUInt32
	traceObservationPoint proto.ColEnum

	interfaceIndex proto.ColUInt32
	interfaceName  proto.ColLowCardinality[string]

	proxyPort proto.ColUInt32
	traceID   proto.ColStr

	sockXLatePoint proto.ColEnum
	socketCookie   proto.ColUInt64
	cgroupID       proto.ColUInt64

	ethernetSrc proto.ColLowCardinality[string]
	ethernetDst proto.ColLowCardinality[string]

	ipv4Src     proto.ColIPv4
	ipv4Dst     proto.ColIPv4
	ipv6Src     proto.ColIPv6
	ipv6Dst     proto.ColIPv6
	ipVersion   proto.ColEnum
	ipEncrypted proto.ColBool

	l4Protocol proto.ColEnum
	l4SrcPort  proto.ColUInt32
	l4DstPort  proto.ColUInt32

	l4TCPFlags proto.ColArr[string]

	l4ICMPType proto.ColUInt32
	l4ICMPCode proto.ColUInt32

	l7FlowType  proto.ColEnum
	l7Protocol  proto.ColEnum
	l7LatencyNs proto.ColUInt64

	l7DNSQuery             proto.ColStr
	l7DNSTTL               proto.ColUInt32
	l7DNSResponseCode      proto.ColUInt16
	l7DNSResponseIPs       proto.ColArr[string]
	l7DNSResponseCNAMEs    proto.ColArr[string]
	l7DNSQTypes            proto.ColArr[string]
	l7DNSRRTypes           proto.ColArr[string]
	l7DNSObservationSource proto.ColStr

	l7HTTPCode          proto.ColUInt16
	l7HTTPMethod        proto.ColLowCardinality[string]
	l7HTTPURL           proto.ColStr
	l7HTTPProtocol      proto.ColLowCardinality[string]
	l7HTTPHeadersKeys   proto.ColArr[string]
	l7HTTPHeadersValues proto.ColArr[string]

	direction proto.ColEnum

	pfmEnv     proto.ColEnum
	pfmNs      proto.ColLowCardinality[string]
	pfmCommit  proto.ColLowCardinality[string]
	pfmUnit    proto.ColLowCardinality[string]
	pfmApp     proto.ColLowCardinality[string]
	pfmProject proto.ColLowCardinality[string]
	pfmDC      proto.ColLowCardinality[string]
	pfmRegion  proto.ColLowCardinality[string]
	pfmCluster proto.ColLowCardinality[string]
	pfmHost    proto.ColLowCardinality[string]

	k8sPod proto.ColLowCardinality[string]
	k8sNS  proto.ColLowCardinality[string]

	pfmPeerEnv     proto.ColEnum
	pfmPeerNs      proto.ColLowCardinality[string]
	pfmPeerCommit  proto.ColLowCardinality[string]
	pfmPeerUnit    proto.ColLowCardinality[string]
	pfmPeerApp     proto.ColLowCardinality[string]
	pfmPeerProject proto.ColLowCardinality[string]
	pfmPeerDC      proto.ColLowCardinality[string]
	pfmPeerRegion  proto.ColLowCardinality[string]
	pfmPeerCluster proto.ColLowCardinality[string]
	pfmPeerHost    proto.ColLowCardinality[string]

	k8sPeerPod proto.ColLowCardinality[string]
	k8sPeerNS  proto.ColLowCardinality[string]

	timestamp proto.ColDateTime64
}

func (t *Table) Reset() {
	for _, v := range []proto.Column{
		&t.flowType,
		&t.verdict,
		&t.dropReason,
		&t.nodeName,
		&t.isReply,

		&t.srcNames,
		&t.dstNames,

		&t.eventType,
		&t.eventSubType,

		&t.endpointSrcID,
		&t.endpointSrcIdentity,
		&t.endpointSrcNamespace,
		&t.endpointSrcPodName,
		&t.endpointSrcLabels,
		&t.endpointSrcWorkloadsNames,
		&t.endpointSrcWorkloadsKinds,

		&t.endpointDstID,
		&t.endpointDstIdentity,
		&t.endpointDstNamespace,
		&t.endpointDstPodName,
		&t.endpointDstLabels,
		&t.endpointDstWorkloadsNames,
		&t.endpointDstWorkloadsKinds,

		&t.trafficDirection,
		&t.policyMatchType,
		&t.traceObservationPoint,

		&t.interfaceIndex,
		&t.interfaceName,

		&t.proxyPort,
		&t.traceID,

		&t.sockXLatePoint,
		&t.socketCookie,
		&t.cgroupID,

		&t.ethernetSrc,
		&t.ethernetDst,

		&t.ipv4Src,
		&t.ipv4Dst,
		&t.ipv6Src,
		&t.ipv6Dst,
		&t.ipVersion,
		&t.ipEncrypted,

		&t.l4Protocol,
		&t.l4SrcPort,
		&t.l4DstPort,
		&t.l4TCPFlags,

		&t.l4ICMPType,
		&t.l4ICMPCode,

		&t.l7FlowType,
		&t.l7Protocol,
		&t.l7LatencyNs,

		&t.l7DNSQuery,
		&t.l7DNSTTL,
		&t.l7DNSResponseCode,
		&t.l7DNSResponseIPs,
		&t.l7DNSResponseCNAMEs,
		&t.l7DNSQTypes,
		&t.l7DNSRRTypes,
		&t.l7DNSObservationSource,

		&t.l7HTTPCode,
		&t.l7HTTPMethod,
		&t.l7HTTPURL,
		&t.l7HTTPProtocol,
		&t.l7HTTPHeadersKeys,
		&t.l7HTTPHeadersValues,

		&t.direction,

		&t.pfmEnv,
		&t.pfmNs,
		&t.pfmCommit,
		&t.pfmUnit,
		&t.pfmApp,
		&t.pfmProject,
		&t.pfmDC,
		&t.pfmRegion,
		&t.pfmCluster,
		&t.pfmHost,
		&t.k8sPod,
		&t.k8sNS,

		&t.pfmPeerEnv,
		&t.pfmPeerNs,
		&t.pfmPeerCommit,
		&t.pfmPeerUnit,
		&t.pfmPeerApp,
		&t.pfmPeerProject,
		&t.pfmPeerDC,
		&t.pfmPeerRegion,
		&t.pfmPeerCluster,
		&t.pfmPeerHost,
		&t.k8sPeerPod,
		&t.k8sPeerNS,
		&t.timestamp,
	} {
		v.Reset()
	}
}

func (t *Table) Rows() int {
	return t.timestamp.Rows()
}

func (t *Table) Insert() string {
	return t.Input().Into(t.name)
}

func (t *Table) Result() proto.Results {
	return proto.Results{
		{Name: "flow_type", Data: &t.flowType},
		{Name: "verdict", Data: &t.verdict},
		{Name: "drop_reason", Data: &t.dropReason},
		{Name: "node_name", Data: &t.nodeName},
		{Name: "is_reply", Data: &t.isReply},

		{Name: "src_names", Data: &t.srcNames},
		{Name: "dst_names", Data: &t.dstNames},

		{Name: "event_type", Data: &t.eventType},
		{Name: "event_sub_type", Data: &t.eventSubType},

		{Name: "endpoint_src_id", Data: &t.endpointSrcID},
		{Name: "endpoint_src_identity", Data: &t.endpointSrcIdentity},
		{Name: "endpoint_src_namespace", Data: &t.endpointSrcNamespace},
		{Name: "endpoint_src_pod_name", Data: &t.endpointSrcPodName},
		{Name: "endpoint_src_labels", Data: &t.endpointSrcLabels},
		{Name: "endpoint_src_workloads_names", Data: &t.endpointSrcWorkloadsNames},
		{Name: "endpoint_src_workloads_kinds", Data: &t.endpointSrcWorkloadsKinds},

		{Name: "endpoint_dst_id", Data: &t.endpointDstID},
		{Name: "endpoint_dst_identity", Data: &t.endpointDstIdentity},
		{Name: "endpoint_dst_namespace", Data: &t.endpointDstNamespace},
		{Name: "endpoint_dst_pod_name", Data: &t.endpointDstPodName},
		{Name: "endpoint_dst_labels", Data: &t.endpointDstLabels},
		{Name: "endpoint_dst_workloads_names", Data: &t.endpointDstWorkloadsNames},
		{Name: "endpoint_dst_workloads_kinds", Data: &t.endpointDstWorkloadsKinds},

		{Name: "traffic_direction", Data: &t.trafficDirection},
		{Name: "policy_match_type", Data: &t.policyMatchType},
		{Name: "trace_observation_point", Data: &t.traceObservationPoint},

		{Name: "interface_index", Data: &t.interfaceIndex},
		{Name: "interface_name", Data: &t.interfaceName},

		{Name: "proxy_port", Data: &t.proxyPort},
		{Name: "trace_id", Data: &t.traceID},

		{Name: "sock_xlate_point", Data: &t.sockXLatePoint},
		{Name: "socket_cookie", Data: &t.socketCookie},
		{Name: "cgroup_id", Data: &t.cgroupID},

		{Name: "ethernet_src", Data: &t.ethernetSrc},
		{Name: "ethernet_dst", Data: &t.ethernetDst},
		{Name: "ipv4_src", Data: &t.ipv4Src},
		{Name: "ipv4_dst", Data: &t.ipv4Dst},
		{Name: "ipv6_src", Data: &t.ipv6Src},
		{Name: "ipv6_dst", Data: &t.ipv6Dst},
		{Name: "ip_version", Data: &t.ipVersion},
		{Name: "ip_encrypted", Data: &t.ipEncrypted},
		{Name: "l4_protocol", Data: &t.l4Protocol},
		{Name: "l4_src_port", Data: &t.l4SrcPort},
		{Name: "l4_dst_port", Data: &t.l4DstPort},
		{Name: "l4_tcp_flags", Data: &t.l4TCPFlags},
		{Name: "l4_icmp_type", Data: &t.l4ICMPType},
		{Name: "l4_icmp_code", Data: &t.l4ICMPCode},

		{Name: "l7_flow_type", Data: &t.l7FlowType},
		{Name: "l7_protocol", Data: &t.l7Protocol},
		{Name: "l7_latency_ns", Data: &t.l7LatencyNs},
		{Name: "l7_dns_query", Data: &t.l7DNSQuery},
		{Name: "l7_dns_ttl", Data: &t.l7DNSTTL},
		{Name: "l7_dns_response_code", Data: &t.l7DNSResponseCode},
		{Name: "l7_dns_response_ips", Data: &t.l7DNSResponseIPs},
		{Name: "l7_dns_response_cnames", Data: &t.l7DNSResponseCNAMEs},
		{Name: "l7_dns_qtypes", Data: &t.l7DNSQTypes},
		{Name: "l7_dns_rrtypes", Data: &t.l7DNSRRTypes},
		{Name: "l7_dns_observation_source", Data: &t.l7DNSObservationSource},

		{Name: "l7_http_code", Data: &t.l7HTTPCode},
		{Name: "l7_http_method", Data: &t.l7HTTPMethod},
		{Name: "l7_http_url", Data: &t.l7HTTPURL},
		{Name: "l7_http_protocol", Data: &t.l7HTTPProtocol},
		{Name: "l7_http_headers_keys", Data: &t.l7HTTPHeadersKeys},
		{Name: "l7_http_headers_values", Data: &t.l7HTTPHeadersValues},

		{Name: "direction", Data: &t.direction},

		{Name: "k8s_pod", Data: &t.k8sPod},
		{Name: "k8s_ns", Data: &t.k8sNS},

		{Name: "k8s_peer_pod", Data: &t.k8sPeerPod},
		{Name: "k8s_peer_ns", Data: &t.k8sPeerNS},

		{Name: "timestamp", Data: &t.timestamp},
	}
}

func (t *Table) ResultColumns() []string {
	var columns []string
	for _, v := range t.Result() {
		name := v.Name
		columns = append(columns, name)
	}
	return columns
}

func (t *Table) Input() proto.Input {
	return proto.Input{
		{Name: "flow_type", Data: &t.flowType},
		{Name: "verdict", Data: &t.verdict},
		{Name: "drop_reason", Data: &t.dropReason},
		{Name: "node_name", Data: &t.nodeName},
		{Name: "is_reply", Data: &t.isReply},

		{Name: "src_names", Data: &t.srcNames},
		{Name: "dst_names", Data: &t.dstNames},

		{Name: "event_type", Data: &t.eventType},
		{Name: "event_sub_type", Data: &t.eventSubType},

		{Name: "endpoint_src_id", Data: &t.endpointSrcID},
		{Name: "endpoint_src_identity", Data: &t.endpointSrcIdentity},
		{Name: "endpoint_src_namespace", Data: &t.endpointSrcNamespace},
		{Name: "endpoint_src_pod_name", Data: &t.endpointSrcPodName},
		{Name: "endpoint_src_labels", Data: &t.endpointSrcLabels},
		{Name: "endpoint_src_workloads_names", Data: &t.endpointSrcWorkloadsNames},
		{Name: "endpoint_src_workloads_kinds", Data: &t.endpointSrcWorkloadsKinds},

		{Name: "endpoint_dst_id", Data: &t.endpointDstID},
		{Name: "endpoint_dst_identity", Data: &t.endpointDstIdentity},
		{Name: "endpoint_dst_namespace", Data: &t.endpointDstNamespace},
		{Name: "endpoint_dst_pod_name", Data: &t.endpointDstPodName},
		{Name: "endpoint_dst_labels", Data: &t.endpointDstLabels},
		{Name: "endpoint_dst_workloads_names", Data: &t.endpointDstWorkloadsNames},
		{Name: "endpoint_dst_workloads_kinds", Data: &t.endpointDstWorkloadsKinds},

		{Name: "traffic_direction", Data: &t.trafficDirection},
		{Name: "policy_match_type", Data: &t.policyMatchType},
		{Name: "trace_observation_point", Data: &t.traceObservationPoint},

		{Name: "interface_index", Data: &t.interfaceIndex},
		{Name: "interface_name", Data: &t.interfaceName},

		{Name: "proxy_port", Data: &t.proxyPort},
		{Name: "trace_id", Data: &t.traceID},

		{Name: "sock_xlate_point", Data: &t.sockXLatePoint},
		{Name: "socket_cookie", Data: &t.socketCookie},
		{Name: "cgroup_id", Data: &t.cgroupID},

		{Name: "ethernet_src", Data: &t.ethernetSrc},
		{Name: "ethernet_dst", Data: &t.ethernetDst},
		{Name: "ipv4_src", Data: &t.ipv4Src},
		{Name: "ipv4_dst", Data: &t.ipv4Dst},
		{Name: "ipv6_src", Data: &t.ipv6Src},
		{Name: "ipv6_dst", Data: &t.ipv6Dst},
		{Name: "ip_version", Data: &t.ipVersion},
		{Name: "ip_encrypted", Data: &t.ipEncrypted},
		{Name: "l4_protocol", Data: &t.l4Protocol},
		{Name: "l4_src_port", Data: &t.l4SrcPort},
		{Name: "l4_dst_port", Data: &t.l4DstPort},
		{Name: "l4_tcp_flags", Data: &t.l4TCPFlags},
		{Name: "l4_icmp_type", Data: &t.l4ICMPType},
		{Name: "l4_icmp_code", Data: &t.l4ICMPCode},

		{Name: "l7_flow_type", Data: &t.l7FlowType},
		{Name: "l7_protocol", Data: &t.l7Protocol},
		{Name: "l7_latency_ns", Data: &t.l7LatencyNs},
		{Name: "l7_dns_query", Data: &t.l7DNSQuery},
		{Name: "l7_dns_ttl", Data: &t.l7DNSTTL},
		{Name: "l7_dns_response_code", Data: &t.l7DNSResponseCode},
		{Name: "l7_dns_response_ips", Data: &t.l7DNSResponseIPs},
		{Name: "l7_dns_response_cnames", Data: &t.l7DNSResponseCNAMEs},
		{Name: "l7_dns_qtypes", Data: &t.l7DNSQTypes},
		{Name: "l7_dns_rrtypes", Data: &t.l7DNSRRTypes},
		{Name: "l7_dns_observation_source", Data: &t.l7DNSObservationSource},

		{Name: "l7_http_code", Data: &t.l7HTTPCode},
		{Name: "l7_http_method", Data: &t.l7HTTPMethod},
		{Name: "l7_http_url", Data: &t.l7HTTPURL},
		{Name: "l7_http_protocol", Data: &t.l7HTTPProtocol},
		{Name: "l7_http_headers_keys", Data: &t.l7HTTPHeadersKeys},
		{Name: "l7_http_headers_values", Data: &t.l7HTTPHeadersValues},

		{Name: "direction", Data: &t.direction},

		{Name: "k8s_pod", Data: &t.k8sPod},
		{Name: "k8s_ns", Data: &t.k8sNS},

		{Name: "k8s_peer_pod", Data: &t.k8sPeerPod},
		{Name: "k8s_peer_ns", Data: &t.k8sPeerNS},

		{Name: "timestamp", Data: &t.timestamp},
	}
}

func (t *Table) Each(fn func(row Row) error) error {
	for i := 0; i < t.Rows(); i++ {
		f := &observer.Flow{
			Type:             flow.FlowType(flow.FlowType_value[t.flowType.Row(i)]),
			Verdict:          flow.Verdict(flow.Verdict_value[t.verdict.Row(i)]),
			DropReasonDesc:   flow.DropReason(flow.DropReason_value[t.dropReason.Row(i)]),
			NodeName:         t.nodeName.Row(i),
			SourceNames:      t.srcNames.Row(i),
			DestinationNames: t.dstNames.Row(i),

			EventType: &observer.CiliumEventType{
				Type:    t.eventType.Row(i),
				SubType: t.eventSubType.Row(i),
			},

			PolicyMatchType: t.policyMatchType.Row(i),

			TraceObservationPoint: flow.TraceObservationPoint(flow.TraceObservationPoint_value[t.traceObservationPoint.Row(i)]),
			TrafficDirection:      flow.TrafficDirection(flow.TrafficDirection_value[t.trafficDirection.Row(i)]),

			ProxyPort:      t.proxyPort.Row(i),
			SocketCookie:   t.socketCookie.Row(i),
			SockXlatePoint: flow.SocketTranslationPoint(flow.SocketTranslationPoint_value[t.sockXLatePoint.Row(i)]),
			CgroupId:       t.cgroupID.Row(i),

			Time: timestamppb.New(t.timestamp.Row(i)),
		}
		if v := t.isReply.Row(i); v.Set {
			f.IsReply = wrapperspb.Bool(v.Value)
		} else {
			f.IsReply = nil
		}

		if name, id := t.interfaceName.Row(i), t.interfaceIndex.Row(i); name != "" || id != 0 {
			f.Interface = &observer.NetworkInterface{
				Name:  name,
				Index: id,
			}
		}
		if id := t.traceID.Row(i); id != "" {
			f.TraceContext = &observer.TraceContext{
				Parent: &observer.TraceParent{
					TraceId: id,
				},
			}
		}

		if t.endpointSrcIdentity.Row(i) != 0 {
			f.Source = &observer.Endpoint{
				ID:        t.endpointSrcID.Row(i),
				Identity:  t.endpointSrcIdentity.Row(i),
				Namespace: t.endpointSrcNamespace.Row(i),
				Labels:    t.endpointSrcLabels.Row(i),
				PodName:   t.endpointSrcPodName.Row(i),
			}
			if len(t.endpointSrcWorkloadsNames.Row(i)) > 0 {
				var workloads []*observer.Workload
				names := t.endpointSrcWorkloadsNames.Row(i)
				kinds := t.endpointSrcWorkloadsKinds.Row(i)
				for k := range names {
					workloads = append(workloads, &observer.Workload{
						Name: names[k],
						Kind: kinds[k],
					})
				}
				f.Source.Workloads = workloads
			}
		}
		if t.endpointDstIdentity.Row(i) != 0 {
			f.Destination = &observer.Endpoint{
				ID:        t.endpointDstID.Row(i),
				Identity:  t.endpointDstIdentity.Row(i),
				Namespace: t.endpointDstNamespace.Row(i),
				Labels:    t.endpointDstLabels.Row(i),
				PodName:   t.endpointDstPodName.Row(i),
			}
			if len(t.endpointDstWorkloadsNames.Row(i)) > 0 {
				var workloads []*observer.Workload
				names := t.endpointDstWorkloadsNames.Row(i)
				kinds := t.endpointDstWorkloadsKinds.Row(i)
				for k := range names {
					workloads = append(workloads, &observer.Workload{
						Name: names[k],
						Kind: kinds[k],
					})
				}
				f.Destination.Workloads = workloads
			}
		}

		var (
			ethernetSrc = t.ethernetSrc.Row(i)
			ethernetDst = t.ethernetDst.Row(i)
		)
		if ethernetSrc != "" || ethernetDst != "" {
			f.Ethernet = &observer.Ethernet{
				Source:      ethernetSrc,
				Destination: ethernetDst,
			}
		}

		switch t.ipVersion.Row(i) {
		case "IPv4":
			f.IP = &observer.IP{
				Source:      t.ipv4Src.Row(i).String(),
				Destination: t.ipv4Dst.Row(i).String(),
				IpVersion:   observer.IPVersion_IPv4,
				Encrypted:   t.ipEncrypted.Row(i),
			}
		case "IPv6":
			f.IP = &observer.IP{
				Source:      t.ipv6Src.Row(i).String(),
				Destination: t.ipv6Dst.Row(i).String(),
				IpVersion:   observer.IPVersion_IPv6,
				Encrypted:   t.ipEncrypted.Row(i),
			}
		}

		switch t.l4Protocol.Row(i) {
		case "TCP":
			flags := &observer.TCPFlags{}
			for _, v := range t.l4TCPFlags.Row(i) {
				switch v {
				case "SYN":
					flags.SYN = true
				case "ACK":
					flags.ACK = true
				case "FIN":
					flags.FIN = true
				case "RST":
					flags.RST = true
				case "URG":
					flags.URG = true
				case "ECE":
					flags.ECE = true
				case "CWR":
					flags.CWR = true
				case "NS":
					flags.NS = true
				}
			}
			f.L4 = &observer.Layer4{
				Protocol: &observer.Layer4_TCP{
					TCP: &observer.TCP{
						Flags:           flags,
						SourcePort:      t.l4SrcPort.Row(i),
						DestinationPort: t.l4DstPort.Row(i),
					},
				},
			}
		case "UDP":
			f.L4 = &observer.Layer4{
				Protocol: &observer.Layer4_UDP{
					UDP: &observer.UDP{
						SourcePort:      t.l4SrcPort.Row(i),
						DestinationPort: t.l4DstPort.Row(i),
					},
				},
			}
		case "SCTP":
			f.L4 = &observer.Layer4{
				Protocol: &observer.Layer4_SCTP{
					SCTP: &observer.SCTP{
						SourcePort:      t.l4SrcPort.Row(i),
						DestinationPort: t.l4DstPort.Row(i),
					},
				},
			}
		case "ICMPv4":
			f.L4 = &observer.Layer4{
				Protocol: &observer.Layer4_ICMPv4{
					ICMPv4: &observer.ICMPv4{
						Type: t.l4ICMPType.Row(i),
						Code: t.l4ICMPCode.Row(i),
					},
				},
			}
		case "ICMPv6":
			f.L4 = &observer.Layer4{
				Protocol: &observer.Layer4_ICMPv6{
					ICMPv6: &observer.ICMPv6{
						Type: t.l4ICMPType.Row(i),
						Code: t.l4ICMPCode.Row(i),
					},
				},
			}
		}

		if p := t.l7Protocol.Row(i); p != "UNKNOWN" && p != "" {
			f.L7 = &observer.Layer7{
				LatencyNs: t.l7LatencyNs.Row(i),
				Type:      flow.L7FlowType(flow.L7FlowType_value[t.l7FlowType.Row(i)]),
			}
			switch t.l7Protocol.Row(i) {
			case "HTTP":
				var headers []*observer.HTTPHeader
				keys, values := t.l7HTTPHeadersKeys.Row(i), t.l7HTTPHeadersValues.Row(i)
				for k := range keys {
					headers = append(headers, &observer.HTTPHeader{
						Key:   keys[k],
						Value: values[k],
					})
				}
				f.L7.Record = &observer.Layer7_Http{
					Http: &observer.HTTP{
						Method:   t.l7HTTPMethod.Row(i),
						Url:      t.l7HTTPURL.Row(i),
						Protocol: t.l7HTTPProtocol.Row(i),
						Code:     uint32(t.l7HTTPCode.Row(i)),
						Headers:  headers,
					},
				}
			case "DNS":
				f.L7.Record = &observer.Layer7_Dns{
					Dns: &observer.DNS{
						Query:             t.l7DNSQuery.Row(i),
						Ttl:               t.l7DNSTTL.Row(i),
						Rcode:             uint32(t.l7DNSResponseCode.Row(i)),
						Ips:               t.l7DNSResponseIPs.Row(i),
						Cnames:            t.l7DNSResponseCNAMEs.Row(i),
						Rrtypes:           t.l7DNSQTypes.Row(i),
						ObservationSource: t.l7DNSObservationSource.Row(i),
						Qtypes:            t.l7DNSQTypes.Row(i),
					},
				}
			}
		}

		row := Row{
			Raw:     f,
			Inverse: t.direction.Row(i) == "INVERSE",
			Index: Peer{
				Kubernetes: RowKubernetes{
					Pod:       t.k8sPod.Row(i),
					Namespace: t.k8sNS.Row(i),
				},
			},
			Peer: Peer{
				Kubernetes: RowKubernetes{
					Pod:       t.k8sPeerPod.Row(i),
					Namespace: t.k8sPeerNS.Row(i),
				},
			},
		}
		if err := fn(row); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

type RowKubernetes struct {
	Pod       string
	Namespace string
	Image     string
	Container string
}

func (t *Table) Append(row Row) error {
	f := row.Raw

	if row.Inverse {
		t.direction.Append("INVERSE")
	} else {
		t.direction.Append("DIRECT")
	}

	t.flowType.Append(f.GetType().String())
	t.verdict.Append(f.GetVerdict().String())
	t.dropReason.Append(f.GetDropReasonDesc().String())
	t.nodeName.Append(f.GetNodeName())
	if f.IsReply == nil {
		t.isReply.Append(proto.Nullable[bool]{})
	} else {
		t.isReply.Append(proto.Nullable[bool]{
			Value: f.GetIsReply().GetValue(),
			Set:   true,
		})
	}

	t.srcNames.Append(f.GetSourceNames())
	t.dstNames.Append(f.GetDestinationNames())

	t.eventType.Append(f.GetEventType().GetType())
	t.eventSubType.Append(f.GetEventType().GetSubType())
	t.policyMatchType.Append(f.GetPolicyMatchType())

	t.proxyPort.Append(f.GetProxyPort())
	t.socketCookie.Append(f.GetSocketCookie())
	t.sockXLatePoint.Append(f.GetSockXlatePoint().String())
	t.cgroupID.Append(f.GetCgroupId())

	t.interfaceIndex.Append(f.GetInterface().GetIndex())
	t.interfaceName.Append(f.GetInterface().GetName())

	t.traceID.Append(f.GetTraceContext().GetParent().GetTraceId())

	eSrc := f.GetSource()
	t.endpointSrcID.Append(eSrc.GetID())
	t.endpointSrcIdentity.Append(eSrc.GetIdentity())
	t.endpointSrcNamespace.Append(eSrc.GetNamespace())
	t.endpointSrcPodName.Append(eSrc.GetPodName())
	t.endpointSrcLabels.Append(eSrc.GetLabels())
	var eSrcNames, eSrcKinds []string
	for _, kv := range eSrc.GetWorkloads() {
		eSrcNames = append(eSrcNames, kv.GetName())
		eSrcKinds = append(eSrcKinds, kv.GetKind())
	}
	t.endpointSrcWorkloadsNames.Append(eSrcNames)
	t.endpointSrcWorkloadsKinds.Append(eSrcKinds)

	eDst := f.GetDestination()
	t.endpointDstID.Append(eDst.GetID())
	t.endpointDstIdentity.Append(eDst.GetIdentity())
	t.endpointDstNamespace.Append(eDst.GetNamespace())
	t.endpointDstPodName.Append(eDst.GetPodName())
	t.endpointDstLabels.Append(eDst.GetLabels())
	var eDstNames, eDstKinds []string
	for _, kv := range eDst.GetWorkloads() {
		eDstNames = append(eDstNames, kv.GetName())
		eDstKinds = append(eDstKinds, kv.GetKind())
	}
	t.endpointDstWorkloadsNames.Append(eDstNames)
	t.endpointDstWorkloadsKinds.Append(eDstKinds)

	t.trafficDirection.Append(f.GetTrafficDirection().String())
	t.traceObservationPoint.Append(f.GetTraceObservationPoint().String())

	t.ethernetSrc.Append(f.GetEthernet().GetSource())
	t.ethernetDst.Append(f.GetEthernet().GetDestination())

	if ip := f.GetIP(); ip.GetIpVersion() != observer.IPVersion_IP_NOT_USED {
		src, err := netip.ParseAddr(ip.GetSource())
		if err != nil {
			return errors.Wrapf(err, "invalid source address: %s", ip.GetSource())
		}
		dst, err := netip.ParseAddr(ip.GetDestination())
		if err != nil {
			return errors.Wrapf(err, "invalid destination address: %s", ip.GetDestination())
		}
		switch ip.GetIpVersion() {
		case observer.IPVersion_IPv4:
			t.ipVersion.Append("IPv4")
			t.ipv4Src.Append(proto.ToIPv4(src))
			t.ipv4Dst.Append(proto.ToIPv4(dst))
			t.ipv6Src.Append(proto.IPv6{})
			t.ipv6Dst.Append(proto.IPv6{})
			t.ipEncrypted.Append(ip.GetEncrypted())
		case observer.IPVersion_IPv6:
			t.ipVersion.Append("IPv6")
			t.ipv4Src.Append(proto.IPv4(0))
			t.ipv4Dst.Append(proto.IPv4(0))
			t.ipv6Src.Append(proto.ToIPv6(src))
			t.ipv6Dst.Append(proto.ToIPv6(dst))
			t.ipEncrypted.Append(ip.GetEncrypted())
		}
	} else {
		t.ipVersion.Append("UNKNOWN")
		t.ipv4Src.Append(proto.IPv4(0))
		t.ipv4Dst.Append(proto.IPv4(0))
		t.ipv6Src.Append(proto.IPv6{})
		t.ipv6Dst.Append(proto.IPv6{})
		t.ipEncrypted.Append(false)
	}

	var (
		srcPort  uint32
		dstPort  uint32
		icmpType uint32
		icmpCode uint32
	)

	l4Protocol := "UNKNOWN"
	if l4 := f.GetL4(); l4 != nil {
		if v := l4.GetTCP(); v != nil {
			srcPort = v.GetSourcePort()
			dstPort = v.GetDestinationPort()
			l4Protocol = "TCP"
		}
		if v := l4.GetUDP(); v != nil {
			srcPort = v.GetSourcePort()
			dstPort = v.GetDestinationPort()
			l4Protocol = "UDP"
		}
		if v := l4.GetICMPv4(); v != nil {
			l4Protocol = "ICMPv4"
			icmpType = v.GetType()
			icmpCode = v.GetCode()
		}
		if v := l4.GetICMPv6(); v != nil {
			l4Protocol = "ICMPv6"
			icmpType = v.GetType()
			icmpCode = v.GetCode()
		}
		if v := l4.GetSCTP(); v != nil {
			srcPort = v.GetSourcePort()
			dstPort = v.GetDestinationPort()
			l4Protocol = "SCTP"
		}
	}

	t.l4Protocol.Append(l4Protocol)
	t.l4SrcPort.Append(srcPort)
	t.l4DstPort.Append(dstPort)
	t.l4ICMPType.Append(icmpType)
	t.l4ICMPCode.Append(icmpCode)

	var tcpFlags []string
	if f := f.GetL4().GetTCP().GetFlags(); f != nil {
		if f.SYN {
			tcpFlags = append(tcpFlags, "SYN")
		}
		if f.ACK {
			tcpFlags = append(tcpFlags, "ACK")
		}
		if f.FIN {
			tcpFlags = append(tcpFlags, "FIN")
		}
		if f.RST {
			tcpFlags = append(tcpFlags, "RST")
		}
		if f.PSH {
			tcpFlags = append(tcpFlags, "PSH")
		}
		if f.URG {
			tcpFlags = append(tcpFlags, "URG")
		}
		if f.ECE {
			tcpFlags = append(tcpFlags, "ECE")
		}
		if f.CWR {
			tcpFlags = append(tcpFlags, "CWR")
		}
		if f.NS {
			tcpFlags = append(tcpFlags, "NS")
		}
	}
	t.l4TCPFlags.Append(tcpFlags)

	l7 := f.GetL7()
	t.l7FlowType.Append(l7.GetType().String())
	if v := l7.GetHttp(); v != nil {
		t.l7Protocol.Append("HTTP")
	} else if v := l7.GetKafka(); v != nil {
		t.l7Protocol.Append("Kafka")
	} else if v := l7.GetDns(); v != nil {
		t.l7Protocol.Append("DNS")
	} else {
		t.l7Protocol.Append("UNKNOWN")
	}
	t.l7LatencyNs.Append(l7.GetLatencyNs())

	t.l7DNSQuery.Append(l7.GetDns().GetQuery())
	t.l7DNSTTL.Append(l7.GetDns().GetTtl())
	t.l7DNSResponseCode.Append(uint16(l7.GetDns().GetRcode()))
	t.l7DNSResponseIPs.Append(l7.GetDns().GetIps())
	t.l7DNSResponseCNAMEs.Append(l7.GetDns().GetCnames())
	t.l7DNSQTypes.Append(l7.GetDns().GetQtypes())
	t.l7DNSRRTypes.Append(l7.GetDns().GetRrtypes())
	t.l7DNSObservationSource.Append(l7.GetDns().GetObservationSource())

	t.l7HTTPCode.Append(uint16(l7.GetHttp().GetCode()))
	t.l7HTTPMethod.Append(l7.GetHttp().GetMethod())
	t.l7HTTPURL.Append(l7.GetHttp().GetUrl())
	t.l7HTTPProtocol.Append(l7.GetHttp().GetProtocol())

	var httpHeaderKeys, httpHeaderValues []string
	for _, h := range l7.GetHttp().GetHeaders() {
		httpHeaderKeys = append(httpHeaderKeys, h.GetKey())
		httpHeaderValues = append(httpHeaderValues, h.GetValue())
	}
	t.l7HTTPHeadersKeys.Append(httpHeaderKeys)
	t.l7HTTPHeadersValues.Append(httpHeaderValues)

	t.k8sPod.Append(row.Index.Kubernetes.Pod)
	t.k8sNS.Append(row.Index.Kubernetes.Namespace)

	t.k8sPeerPod.Append(row.Peer.Kubernetes.Pod)
	t.k8sPeerNS.Append(row.Peer.Kubernetes.Namespace)

	t.timestamp.Append(f.GetTime().AsTime())

	return nil
}

type Peer struct {
	Kubernetes RowKubernetes
}

type Row struct {
	Raw     *observer.Flow
	Index   Peer
	Peer    Peer
	Inverse bool
}

func newStrLowCardinality() proto.ColLowCardinality[string] {
	return *proto.NewLowCardinality[string](&proto.ColStr{})
}

func NewTable(name string) *Table {
	t := &Table{
		name: name,

		nodeName:   newStrLowCardinality(),
		dropReason: newStrLowCardinality(),

		isReply: *(new(proto.ColBool).Nullable()),

		srcNames: *proto.NewArray[string](new(proto.ColStr)),
		dstNames: *proto.NewArray[string](new(proto.ColStr)),

		interfaceName: newStrLowCardinality(),

		endpointSrcNamespace:      newStrLowCardinality(),
		endpointSrcPodName:        newStrLowCardinality(),
		endpointSrcLabels:         *proto.NewArray[string](new(proto.ColStr).LowCardinality()),
		endpointSrcWorkloadsNames: *proto.NewArray[string](new(proto.ColStr).LowCardinality()),
		endpointSrcWorkloadsKinds: *proto.NewArray[string](new(proto.ColStr).LowCardinality()),

		endpointDstNamespace:      newStrLowCardinality(),
		endpointDstPodName:        newStrLowCardinality(),
		endpointDstLabels:         *proto.NewArray[string](new(proto.ColStr).LowCardinality()),
		endpointDstWorkloadsNames: *proto.NewArray[string](new(proto.ColStr).LowCardinality()),
		endpointDstWorkloadsKinds: *proto.NewArray[string](new(proto.ColStr).LowCardinality()),

		ethernetSrc: newStrLowCardinality(),
		ethernetDst: newStrLowCardinality(),

		pfmNs:      newStrLowCardinality(),
		pfmCommit:  newStrLowCardinality(),
		pfmUnit:    newStrLowCardinality(),
		pfmApp:     newStrLowCardinality(),
		pfmProject: newStrLowCardinality(),
		pfmDC:      newStrLowCardinality(),
		pfmRegion:  newStrLowCardinality(),
		pfmCluster: newStrLowCardinality(),
		pfmHost:    newStrLowCardinality(),

		k8sPod: newStrLowCardinality(),
		k8sNS:  newStrLowCardinality(),

		pfmPeerNs:      newStrLowCardinality(),
		pfmPeerCommit:  newStrLowCardinality(),
		pfmPeerUnit:    newStrLowCardinality(),
		pfmPeerApp:     newStrLowCardinality(),
		pfmPeerProject: newStrLowCardinality(),
		pfmPeerDC:      newStrLowCardinality(),
		pfmPeerRegion:  newStrLowCardinality(),
		pfmPeerCluster: newStrLowCardinality(),
		pfmPeerHost:    newStrLowCardinality(),

		k8sPeerPod: newStrLowCardinality(),
		k8sPeerNS:  newStrLowCardinality(),

		l7DNSResponseIPs:    *proto.NewArray[string](new(proto.ColStr)),
		l7DNSResponseCNAMEs: *proto.NewArray[string](new(proto.ColStr)),
		l7DNSQTypes:         *proto.NewArray[string](new(proto.ColStr)),
		l7DNSRRTypes:        *proto.NewArray[string](new(proto.ColStr)),

		l7HTTPMethod:        newStrLowCardinality(),
		l7HTTPProtocol:      newStrLowCardinality(),
		l7HTTPHeadersKeys:   *proto.NewArray[string](new(proto.ColStr).LowCardinality()),
		l7HTTPHeadersValues: *proto.NewArray[string](new(proto.ColStr)),
	}

	t.timestamp.WithPrecision(9)

	t.l4TCPFlags = *proto.NewArray[string](&proto.ColEnum{})

	return t
}
