package sec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/cht"
	"github.com/go-faster/tetragon/api/v1/tetragon"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTable_ResultColumns(t *testing.T) {
	d := NewTable("sec")
	cols := d.ResultColumns()
	inputs := d.Input()
	ddl := NewDDL("sec")

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
	d := NewTable("sec")

	row := Row{
		Res: &tetragon.GetEventsResponse{
			Event: &tetragon.GetEventsResponse_ProcessExec{
				ProcessExec: &tetragon.ProcessExec{
					Process: &tetragon.Process{
						Pod: &tetragon.Pod{
							PodLabels: map[string]string{},
						},
					},
					Parent: &tetragon.Process{},
				},
			},
			NodeName: "node-name",
			Time:     timestamppb.Now(),
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
	q := fmt.Sprintf(`SELECT %s FROM sec`,
		strings.Join(d.ResultColumns(), ", "),
	)
	require.NoError(t, c.Do(ctx, ch.Query{
		Body:   q,
		Result: d.Result(),
	}), "select")
	require.Equal(t, 10, d.Rows())

	_ = d.Each(func(r Row) error {
		return nil
	})
}
