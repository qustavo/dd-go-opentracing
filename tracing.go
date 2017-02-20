package ddtracer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func defaultHostname() string {
	host, _ := os.Hostname()
	return host
}

var (
	DefaultService  = defaultHostname()
	DefaultResource = "/"
)

var (
	ServiceKeyTag  = "service"
	ResourceKeyTag = "resource"
)

type Tracer struct {
	*tracer.Tracer
	textPropagator *textMapPropagator
}

func NewTracer() opentracing.Tracer {
	return NewTracerTransport(nil)
}

func NewTracerTransport(tr tracer.Transport) opentracing.Tracer {
	t := &Tracer{
		Tracer: tracer.NewTracerTransport(tr),
	}
	t.textPropagator = &textMapPropagator{t}

	return t
}

func (t *Tracer) StartSpan(op string, opts ...opentracing.StartSpanOption) opentracing.Span {
	sso := &opentracing.StartSpanOptions{}
	for _, o := range opts {
		o.Apply(sso)
	}

	return t.startSpanWithOptions(op, sso)
}

func (t *Tracer) startSpanWithOptions(op string, opts *opentracing.StartSpanOptions) opentracing.Span {
	var span *tracer.Span
	for _, ref := range opts.References {
		if ref.Type == opentracing.ChildOfRef {
			if p, ok := ref.ReferencedContext.(*SpanContext); ok {
				span = tracer.NewChildSpanFromContext(op, p.ctx)
			}
		}
	}

	if span == nil {
		span = t.NewRootSpan(op, DefaultService, DefaultResource)
	}

	s := &Span{span}
	for key, value := range opts.Tags {
		s.SetTag(key, value)
	}

	return s

}

func (t *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	sc, ok := sm.(*SpanContext)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}

	span, ok := tracer.SpanFromContext(sc.ctx)
	if !ok {
		return opentracing.ErrInvalidSpanContext
	}

	switch format {
	case opentracing.HTTPHeaders:
		return t.textPropagator.Inject(span, carrier)
	}

	return opentracing.ErrUnsupportedFormat
}

func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.HTTPHeaders:
		return t.textPropagator.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}

type Span struct {
	*tracer.Span
}

func (s *Span) Finish() {
	s.Span.Finish()
}

func (s *Span) FinishWithOptions(opts opentracing.FinishOptions) {
	panic("not implemented")
}

func (s *Span) Context() opentracing.SpanContext {
	ctx := s.Span.Context(context.Background())
	return &SpanContext{ctx}
}

func (s *Span) SetOperationName(operationName string) opentracing.Span {
	s.Name = operationName
	return s
}

func (s *Span) setMeta(key string, value interface{}) opentracing.Span {
	val := fmt.Sprint(value)
	switch key {
	case ServiceKeyTag:
		s.Service = val
	case ResourceKeyTag:
		s.Resource = val
	default:
		s.SetMeta(key, val)
	}

	return s
}

func (s *Span) SetTag(key string, value interface{}) opentracing.Span {
	switch t := value.(type) {
	case float64:
		s.SetMetric(key, t)
	default:
		s.setMeta(key, fmt.Sprint(value))
	}

	return s
}

func (s *Span) LogFields(fields ...log.Field) {
	for _, field := range fields {
		s.SetTag(field.Key(), field.Value())
	}
}

func (s *Span) LogKV(alternatingKeyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(alternatingKeyValues...)
	if err != nil {
		return
	}
	s.LogFields(fields...)
}

func (s *Span) SetBaggageItem(restrictedKey string, value string) opentracing.Span {
	panic("not implemented")
}

func (s *Span) BaggageItem(restrictedKey string) string {
	panic("not implemented")
}

func (s *Span) Tracer() opentracing.Tracer {
	return s.Tracer()
}

func (s *Span) LogEvent(event string) {
	panic("not implemented")
}

func (s *Span) LogEventWithPayload(event string, payload interface{}) {
	panic("not implemented")
}

func (s *Span) Log(data opentracing.LogData) {
	panic("not implemented")
}

type SpanContext struct {
	ctx context.Context
}

func (ctx *SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	panic("not implemented")
}
