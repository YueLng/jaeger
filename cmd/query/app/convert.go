package app

import (
	"time"

	m "github.com/jaegertracing/jaeger/model"
	j "github.com/uber/jaeger-client-go/thrift-gen/jaeger"
)

func convertThriftToModel(spans []*j.Span, process *j.Process) []m.Span {
	mspans := []m.Span{}
	for _, s := range spans {

		t := m.Span{
			TraceID:       newTraceID(s.TraceIdLow, s.TraceIdHigh),
			SpanID:        m.SpanID(s.SpanId),
			ParentSpanID:  m.SpanID(s.ParentSpanId),
			OperationName: s.OperationName,
			References:    convertReference(s.References),
			Flags:         m.Flags(s.Flags),
			StartTime:     time.Unix(0, s.StartTime*int64(time.Microsecond)),
			Duration:      time.Microsecond * time.Duration(s.Duration),
			Tags:          convertTags(s.Tags),
			Logs:          convertLogs(s.Logs),
			Process:       convertProcess(process),
		}
		mspans = append(mspans, t)
	}
	return mspans
}

func newTraceID(low, high int64) m.TraceID {
	return m.TraceID{
		Low:  uint64(low),
		High: uint64(high),
	}
}

func convertReference(references []*j.SpanRef) []m.SpanRef {
	var mSpanRef []m.SpanRef
	for _, r := range references {
		mSpanRef = append(mSpanRef, m.SpanRef{
			RefType: m.SpanRefType(r.RefType),
			TraceID: newTraceID(r.TraceIdLow, r.TraceIdHigh),
			SpanID:  m.SpanID(r.SpanId),
		})
	}
	return mSpanRef
}

func convertTags(tags []*j.Tag) m.KeyValues {
	var kvs m.KeyValues
	for _, t := range tags {
		kv := m.KeyValue{}
		switch t.VType {
		case j.TagType_STRING:
			kv = m.String(t.Key, *t.VStr)
		case j.TagType_BOOL:
			kv = m.Bool(t.Key, *t.VBool)
		case j.TagType_LONG:
			kv = m.Int64(t.Key, *t.VLong)
		case j.TagType_BINARY:
			kv = m.Binary(t.Key, t.VBinary)
		case j.TagType_DOUBLE:
			kv = m.Float64(t.Key, *t.VDouble)
		}
		kvs = append(kvs, kv)
	}
	return kvs
}

func convertLogs(logs []*j.Log) []m.Log {
	var mlogs []m.Log
	for _, l := range logs {
		mlogs = append(mlogs, m.Log{
			Timestamp: time.Unix(l.Timestamp, 0),
			Fields:    convertTags(l.Fields),
		})
	}
	return mlogs
}

func convertProcess(p *j.Process) *m.Process {
	return &m.Process{
		ServiceName: p.ServiceName,
		Tags:        convertTags(p.Tags),
	}
}

func convertProcesss(process []*j.Process) []m.Process {
	var mprocess []m.Process
	for _, p := range process {
		mprocess = append(mprocess, m.Process{
			ServiceName: p.ServiceName,
			Tags:        convertTags(p.Tags),
		})
	}
	return mprocess
}
