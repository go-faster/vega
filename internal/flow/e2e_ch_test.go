package flow

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/cht"
	"github.com/cilium/cilium/api/v1/observer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestTable_ResultColumns(t *testing.T) {
	d := NewTable("flows")
	cols := d.ResultColumns()
	inputs := d.Input()
	ddl := NewDDL("flows")

	require.Equal(t, len(cols), len(inputs))
	for i := range cols {
		require.Equal(t, cols[i], inputs[i].Name)
		require.True(t, strings.Contains(ddl, cols[i]))
	}
}

func TestIntegrationClickHouseColumns(t *testing.T) {
	cht.Skip(t)
	s := cht.New(t)
	ctx := context.Background()
	c, err := ch.Dial(ctx, ch.Options{
		Address: s.TCP,
		Logger:  zaptest.NewLogger(t),
	})
	require.NoError(t, err)
	ddl := DDL
	ddl += "\nTTL toDateTime(timestamp) + INTERVAL 6 HOUR"
	require.NoError(t, c.Do(ctx, ch.Query{Body: ddl}), "DDL")
	d := NewTable("flows")

	row := Row{
		Peer: Peer{
			Kubernetes: RowKubernetes{
				Namespace: "ns",
				Pod:       "pod",
			},
		},
		Index: Peer{
			Kubernetes: RowKubernetes{
				Namespace: "index-ns",
				Pod:       "index-pod",
			},
		},
		Inverse: true,
		Raw: &observer.Flow{
			Verdict:               observer.Verdict_FORWARDED,
			Type:                  observer.FlowType_L3_L4,
			TraceObservationPoint: observer.TraceObservationPoint_FROM_HOST,
			Source: &observer.Endpoint{
				ID:       123,
				Identity: 1123414,
				Labels: []string{
					"reserved:host",
				},
			},
			SourceNames: []string{
				"source", "names",
			},
			Destination: &observer.Endpoint{
				ID:       456,
				Identity: 1123415,
				Labels: []string{
					"k8s:foo=bar",
				},
			},
			Interface: &observer.NetworkInterface{
				Name:  "eth0",
				Index: 3,
			},
			TraceContext: &observer.TraceContext{
				Parent: &observer.TraceParent{
					TraceId: "trace-id",
				},
			},

			PolicyMatchType: 203,

			CgroupId:     20041,
			SocketCookie: 23,

			ProxyPort: 8080,

			SockXlatePoint: observer.SocketTranslationPoint_SOCK_XLATE_POINT_POST_DIRECTION_REV,

			DestinationNames: []string{
				"destination", "names",
			},
			NodeName: "node",
			Time:     timestamppb.Now(),
			L4: &observer.Layer4{
				Protocol: &observer.Layer4_TCP{
					TCP: &observer.TCP{
						SourcePort:      1234,
						DestinationPort: 5678,
						Flags: &observer.TCPFlags{
							SYN: true,
							ACK: true,
						},
					},
				},
			},
			IsReply: wrapperspb.Bool(true),
			IP: &observer.IP{
				Source:      "10.0.0.1",
				Destination: "10.0.0.2",
				IpVersion:   observer.IPVersion_IPv4,
				Encrypted:   false,
			},
			Ethernet: &observer.Ethernet{
				Source:      "00:00:00:00:00:01",
				Destination: "00:00:00:00:00:02",
			},
			EventType: &observer.CiliumEventType{
				Type:    1,
				SubType: 10001,
			},
			L7: &observer.Layer7{
				Type:      observer.L7FlowType_REQUEST,
				LatencyNs: 1205001,
				Record: &observer.Layer7_Http{
					Http: &observer.HTTP{
						Method:   "GET",
						Url:      "https://example.com",
						Protocol: "HTTP/1.1",
						Code:     200,
						Headers: []*observer.HTTPHeader{
							{
								Key:   "Host",
								Value: "example.com",
							},
							{
								Key:   "User-Agent",
								Value: "curl/7.64.1",
							},
						},
					},
				},
			},
		},
	}
	for i := 0; i < 20; i++ {
		if i == 10 {
			d.Reset()
		}
		require.NoError(t, d.Append(row))
	}
	require.NoError(t, c.Do(ctx, ch.Query{
		Body:  d.Insert(),
		Input: d.Input(),
	}), "insert")

	// Build select query for log subset.
	q := fmt.Sprintf(`SELECT %s FROM flows`,
		strings.Join(d.ResultColumns(), ", "),
	)
	require.NoError(t, c.Do(ctx, ch.Query{
		Body:   q,
		Result: d.Result(),
	}), "select")
	require.Equal(t, 10, d.Rows())

	_ = d.Each(func(r Row) error {
		require.Equal(t, row.Raw.String(), r.Raw.String())
		require.Equal(t, row.Peer, r.Peer)
		require.Equal(t, row.Inverse, r.Inverse)
		require.Equal(t, row.Index, r.Index)
		return nil
	})
}
