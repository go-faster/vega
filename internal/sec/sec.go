// Package sec implements ClickHouse schema and data ingestion for security
// events from tetragon.
package sec

import (
	"encoding/json"
	"fmt"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/go-faster/errors"
	"github.com/go-faster/tetragon/api/v1/tetragon"
)

func NewDDL(tableName string) string {
	// DDL for ClickHouse table.
	const ddl = `
CREATE TABLE IF NOT EXISTS %s
(
    timestamp                    DateTime64(9),
    -- index for time-based queries
    INDEX timestamp_idx timestamp TYPE minmax GRANULARITY 1,
    -- Name of the node where this event was observed.
    node_name                    LowCardinality(String),

	-- k8s materialized fields
    k8s_pod       LowCardinality(String),
    k8s_container LowCardinality(String),
    k8s_ns        LowCardinality(String),
    k8s_image     LowCardinality(String),

    event_type Enum8(
      'ProcessExec'      = 1,
      'ProcessExit'      = 5,
      'ProcessKprobe'    = 9,
      'ProcessTracepoint'= 10,
      'ProcessLoader'    = 11
    ),

    -- processes fields (0 - process, 1 - parent, 2..N - other ancestors, only for ProcessExec)
    process_exec_id        String,
    process_pid            UInt32,
    process_uid            UInt32,
    process_cwd            String,
    process_binary         String,
    process_args           String,
    process_flags          String,
    process_start_time     DateTime64(9),
    process_auid           UInt32,
    process_docker         String,
    process_parent_exec_id String,
    process_refcnt         UInt32,

    parent_process_exec_id        String,
    parent_process_pid            UInt32,
    parent_process_uid            UInt32,
    parent_process_cwd            String,
    parent_process_binary         String,
    parent_process_args           String,
    parent_process_flags          String,
    parent_process_start_time     DateTime64(9),
    parent_process_auid           UInt32,
    parent_process_docker         String,
    parent_process_parent_exec_id String,
    parent_process_refcnt         UInt32,

    process_ancestors_json String
)
    ENGINE = MergeTree()
        PARTITION BY toYearWeek(timestamp)
        ORDER BY (node_name, timestamp)
`
	return fmt.Sprintf(ddl, tableName)
}

// DDL for ClickHouse table.
var DDL = NewDDL("sec")

type Process struct {
	prefix string

	processExecID       proto.ColStr
	processPID          proto.ColUInt32
	processUID          proto.ColUInt32
	processCWD          proto.ColStr
	processBinary       proto.ColStr
	processArgs         proto.ColStr
	processFlags        proto.ColStr
	processStartTime    proto.ColDateTime64
	processAuid         proto.ColUInt32
	processDocker       proto.ColStr
	processParentExecID proto.ColStr
	processRefcnt       proto.ColUInt32
}

func NewProcess(prefix string) *Process {
	p := &Process{
		prefix: prefix,
	}
	p.processStartTime.WithPrecision(9)
	return p
}

func (p *Process) Append(v *tetragon.Process) {
	p.processExecID.Append(v.GetExecId())
	p.processPID.Append(v.GetPid().GetValue())
	p.processUID.Append(v.GetUid().GetValue())
	p.processCWD.Append(v.GetCwd())
	p.processBinary.Append(v.GetBinary())
	p.processArgs.Append(v.GetArguments())
	p.processFlags.Append(v.GetFlags())
	p.processStartTime.Append(v.GetStartTime().AsTime())
	p.processAuid.Append(v.GetAuid().GetValue())
	p.processDocker.Append(v.GetDocker())
	p.processParentExecID.Append(v.GetParentExecId())
	p.processRefcnt.Append(v.GetRefcnt())
}

func (p *Process) Columns() []Column {
	var out []Column
	for _, v := range []Column{
		{Name: "process_exec_id", Data: &p.processExecID},
		{Name: "process_pid", Data: &p.processPID},
		{Name: "process_uid", Data: &p.processUID},
		{Name: "process_cwd", Data: &p.processCWD},
		{Name: "process_binary", Data: &p.processBinary},
		{Name: "process_args", Data: &p.processArgs},
		{Name: "process_flags", Data: &p.processFlags},
		{Name: "process_start_time", Data: &p.processStartTime},
		{Name: "process_auid", Data: &p.processAuid},
		{Name: "process_docker", Data: &p.processDocker},
		{Name: "process_parent_exec_id", Data: &p.processParentExecID},
		{Name: "process_refcnt", Data: &p.processRefcnt},
	} {
		v.Name = p.prefix + v.Name
		out = append(out, v)
	}

	return out
}

func (p *Process) Reset() {
	for _, v := range p.Columns() {
		v.Data.Reset()
	}
}

func (p *Process) Result() proto.Results {
	var out []proto.ResultColumn
	for _, v := range p.Columns() {
		out = append(out, proto.ResultColumn{
			Name: v.Name,
			Data: v.Data,
		})
	}

	return out
}

func (p *Process) Input() proto.Input {
	var out proto.Input
	for _, v := range p.Columns() {
		out = append(out, proto.InputColumn{
			Name: v.Name,
			Data: v.Data,
		})
	}
	return out
}

type Column struct {
	Name string
	Data proto.Column
}

// Table is wrapper for ClickHouse columns that simplifies data ingestion.
type Table struct {
	name string

	timestamp proto.ColDateTime64
	node      proto.ColLowCardinality[string]
	eventType proto.ColEnum

	k8sPod       proto.ColLowCardinality[string]
	k8sNS        proto.ColLowCardinality[string]
	k8sContainer proto.ColLowCardinality[string]
	k8sImage     proto.ColLowCardinality[string]

	process *Process
	parent  *Process

	ancestors proto.ColStr // json of ancestors
}

func (t *Table) Reset() {
	for _, v := range t.Columns() {
		v.Data.Reset()
	}
}

func (t *Table) Rows() int {
	return t.timestamp.Rows()
}

func (t *Table) Insert() string {
	return t.Input().Into(t.name)
}

func (t *Table) Result() proto.Results {
	var out proto.Results
	for _, v := range t.Columns() {
		out = append(out, proto.ResultColumn{
			Name: v.Name,
			Data: v.Data,
		})
	}
	return out
}

func (t *Table) ResultColumns() []string {
	var columns []string
	for _, v := range t.Result() {
		name := v.Name
		columns = append(columns, name)
	}
	return columns
}

func (t *Table) Columns() []Column {
	c := []Column{
		{Name: "timestamp", Data: &t.timestamp},
		{Name: "node_name", Data: &t.node},

		{Name: "k8s_pod", Data: &t.k8sPod},
		{Name: "k8s_ns", Data: &t.k8sNS},
		{Name: "k8s_container", Data: &t.k8sContainer},
		{Name: "k8s_image", Data: &t.k8sImage},

		{Name: "process_ancestors_json", Data: &t.ancestors},
	}

	c = append(c, t.process.Columns()...)
	c = append(c, t.parent.Columns()...)

	return c
}

func (t *Table) Input() proto.Input {
	var out proto.Input
	for _, v := range t.Columns() {
		out = append(out, proto.InputColumn{
			Name: v.Name,
			Data: v.Data,
		})
	}
	return out
}

func (t *Table) Each(f func(row Row) error) error {
	for i := 0; i < t.Rows(); i++ {
		row := Row{}
		if err := f(row); err != nil {
			return errors.Wrapf(err, "[%d]", i)
		}
	}
	return nil
}

func (t *Table) Append(row Row) error {
	r := row.Res

	switch v := r.Event.(type) {
	case *tetragon.GetEventsResponse_ProcessExec:
		e := v.ProcessExec
		t.process.Append(e.GetProcess())
		t.parent.Append(e.GetParent())

		data, err := json.Marshal(e.GetAncestors())
		if err != nil {
			return errors.Wrap(err, "marshal ancestors")
		}
		t.ancestors.AppendBytes(data)

		pod := e.GetProcess().GetPod()

		t.k8sNS.Append(pod.GetNamespace())
		t.k8sPod.Append(pod.GetName())
		t.k8sContainer.Append(pod.GetContainer().GetName())
		t.k8sImage.Append(pod.GetContainer().GetImage().GetId())

	case *tetragon.GetEventsResponse_ProcessExit:
		e := v.ProcessExit
		t.process.Append(e.GetProcess())
		t.parent.Append(e.GetParent())

		pod := e.GetProcess().GetPod()

		t.k8sNS.Append(pod.GetNamespace())
		t.k8sPod.Append(pod.GetName())
		t.k8sContainer.Append(pod.GetContainer().GetName())
		t.k8sImage.Append(pod.GetContainer().GetImage().GetId())

		t.ancestors.Append("null")
	default:
		return errors.Errorf("unknown event type: %T", r.Event)
	}

	t.timestamp.Append(r.Time.AsTime())
	t.node.Append(r.NodeName)
	t.eventType.Append(r.EventType().String())

	return nil
}

type Row struct {
	Res *tetragon.GetEventsResponse
}

func newStrLowCardinality() proto.ColLowCardinality[string] {
	return *proto.NewLowCardinality[string](&proto.ColStr{})
}

func NewTable(name string) *Table {
	t := &Table{
		name: name,
		node: *proto.NewLowCardinality[string](&proto.ColStr{}),

		k8sPod:       newStrLowCardinality(),
		k8sNS:        newStrLowCardinality(),
		k8sContainer: newStrLowCardinality(),
		k8sImage:     newStrLowCardinality(),

		process: NewProcess(""),
		parent:  NewProcess("parent_"),
	}
	t.timestamp.WithPrecision(9)
	return t
}
