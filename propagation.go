package ddtracer

import (
	"strconv"
	"strings"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	tracePrefix = "dd-trace-"

	fieldSpanID   = tracePrefix + "spanid"
	fieldTraceID  = tracePrefix + "traceid"
	fieldParentID = tracePrefix + "parentid"
	//fieldSampled = tracePrefix + "sampled"
)

type textMapPropagator struct {
	t *Tracer
}

// Inject takes a span and a supplied carrier and injects the span into the carrier
func (p *textMapPropagator) Inject(span *tracer.Span, carrier interface{}) error {
	tm, ok := carrier.(opentracing.TextMapWriter)
	if !ok {
		return opentracing.ErrInvalidCarrier
	}

	tm.Set(fieldSpanID, strconv.FormatUint(span.SpanID, 16))
	tm.Set(fieldTraceID, strconv.FormatUint(span.TraceID, 16))
	if span.ParentID > 0 {
		tm.Set(fieldParentID, strconv.FormatUint(span.ParentID, 16))
	}

	return nil
}

// Extract a full span object from a supplied carrier if one exists
func (p *textMapPropagator) Extract(carrier interface{}) (opentracing.SpanContext, error) {
	tm, ok := carrier.(opentracing.TextMapReader)
	if !ok {
		return nil, opentracing.ErrInvalidCarrier
	}

	var err error
	var spanID, traceID, parentID uint64
	err = tm.ForeachKey(func(k, v string) error {
		switch strings.ToLower(k) {
		case fieldSpanID:
			spanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case fieldTraceID:
			traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		case fieldParentID:
			parentID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return opentracing.ErrSpanContextCorrupted
			}
		}

		return nil
	})

	span := &Span{
		tracer.NewSpan("", DefaultService, DefaultResource, spanID, traceID, parentID, p.t.Tracer),
		p.t,
	}
	return span.Context(), err
}
