package ddtracer

import (
	"context"
	"fmt"
	stdlog "log"
	"math/rand"
	"os"
	"time"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
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
	// DefaultService is set to the hostname by default
	DefaultService = defaultHostname()
	// DefaultResource is the resource name if one isn't given
	DefaultResource = "/"

	// EnvTag set's the environment for a given span
	// i.e EnvTag.Set(span, "development")
	EnvTag = stringTagName("env")
)

// Tracer extends the DataDog tracer supplying our own text propagator
type Tracer struct {
	*tracer.Tracer
	textPropagator *textMapPropagator
}

// NewTracer creates a new Tracer.
func NewTracer() opentracing.Tracer {
	return NewTracerTransport(nil)
}

// NewTracerTransport create a new Tracer with the given transport.
func NewTracerTransport(tr tracer.Transport) opentracing.Tracer {
	var driver *tracer.Tracer
	if tr == nil {
		driver = tracer.NewTracer()
	} else {
		driver = tracer.NewTracerTransport(tr)
	}

	t := &Tracer{Tracer: driver}
	t.textPropagator = &textMapPropagator{t}

	return t
}

// StartSpan creates a new span object initialized with the supplied options
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

				// If this is true we're making a DD opentracing span from
				// a standard open tracing span
				if span.TraceID == 0 {
					span.TraceID = span.SpanID // at the very least set the trace ID, with no other changes this becomes a root span
					pSpan := tracer.SpanFromContextDefault(p.ctx)
					if pSpan.TraceID == 0 { // if the parent span doesn't have a trace ID set it
						pSpan.TraceID = pSpan.SpanID
					}
					if span.ParentID == pSpan.SpanID { // if this is infact our parent inherit its trace id
						span.TraceID = pSpan.TraceID
					}
				}
			}
		}
	}

	if span == nil {
		span = t.NewRootSpan(op, DefaultService, DefaultResource)
	}

	s := &Span{span, t}
	for key, value := range opts.Tags {
		s.SetTag(key, value)
	}

	return s

}

// Inject takes a span and a supplied carrier and injects the span into the carrier
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

// Extract a full span object from a supplied carrier if one exists
func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.HTTPHeaders:
		return t.textPropagator.Extract(carrier)
	}
	return nil, opentracing.ErrUnsupportedFormat
}

// Close shuts down the root tracer as well as the embeded one
func (t *Tracer) Close() error {
	t.Stop()
	t.Tracer.Stop()
	return nil
}

// Span extends the DataDog span adding our own tracer to it
type Span struct {
	*tracer.Span
	tracer *Tracer
}

// Finish closes the span
func (s *Span) Finish() {
	s.FinishWithOptions(opentracing.FinishOptions{})
}

// FinishWithOptions closes the span with the supplied options
func (s *Span) FinishWithOptions(opts opentracing.FinishOptions) {
	if !opts.FinishTime.IsZero() {
		s.Duration = opts.FinishTime.UTC().UnixNano() - s.Start
	}
	s.Span.Finish()
}

// Context returns the span context version of the span
func (s *Span) Context() opentracing.SpanContext {
	ctx := s.Span.Context(context.Background())
	return &SpanContext{ctx}
}

// SetOperationName is a setter function for the operationNAame property of the span
func (s *Span) SetOperationName(operationName string) opentracing.Span {
	s.Name = operationName
	return s
}

func (s *Span) setTag(key string, value interface{}) opentracing.Span {
	val := fmt.Sprint(value)
	switch key {
	case string(ext.PeerService):
		s.Service = val
	case string(ext.Component):
		s.Resource = val
	default:
		s.SetMeta(key, val)
	}

	return s
}

// SetTag adds a key+value pair as a tag to the span
func (s *Span) SetTag(key string, value interface{}) opentracing.Span {
	switch t := value.(type) {
	case float64:
		s.SetMetric(key, t)
	default:
		s.setTag(key, value)
	}

	return s
}

// LogFields adds fields to a span, this is a more extensible way of adding tags
func (s *Span) LogFields(fields ...log.Field) {
	for _, field := range fields {
		switch field.Key() {
		case "error":
			s.SetError(field.Value().(error))
		default:
			s.SetTag(field.Key(), field.Value())
		}
	}
}

// LogKV converts key values to fields to be used with LogFields
func (s *Span) LogKV(alternatingKeyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(alternatingKeyValues...)
	if err != nil {
		return
	}
	s.LogFields(fields...)
}

// LogEvent is a deprecated function for a special LogField
func (s *Span) LogEvent(event string) {
	stdlog.Println("Span.LogEvent() has been deprecated, use LogFields or LogKV")
	s.LogKV(event, nil)
}

// LogEventWithPayload is a deprecated function for a special LogField
func (s *Span) LogEventWithPayload(event string, payload interface{}) {
	stdlog.Println("Span.LogEventWithPayload() has been deprecated, use LogFields or LogKV")
	s.LogKV(event, payload)
}

// Log is a deprecated function for a special LogField
func (s *Span) Log(data opentracing.LogData) {
	stdlog.Println("Span.Log() has been deprecated, use LogFields or LogKV")
}

// SetBaggageItem  hasn't been implemented
func (s *Span) SetBaggageItem(restrictedKey string, value string) opentracing.Span {
	stdlog.Println("WARNING - SetBaggageItem not implemented")
	return s
}

// BaggageItem hasn't been implemented
func (s *Span) BaggageItem(restrictedKey string) string {
	stdlog.Println("WARNING - BaggageItem not implemented")
	return ""
}

// Tracer returns the tracer the span is associated with
func (s *Span) Tracer() opentracing.Tracer {
	return s.tracer
}

// SpanContext is a type used for converting a span into a context obj
type SpanContext struct {
	ctx context.Context
}

// ForeachBaggageItem hasn't been implemented
func (ctx *SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	stdlog.Println("WARNING - ForeachBaggageItem not implemented")
}

type stringTagName string

// Set a value for a tag in a supplied span
func (tag stringTagName) Set(span opentracing.Span, value string) {
	span.SetTag(string(tag), value)
}
