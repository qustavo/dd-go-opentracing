package ddtracer

import (
	"io"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
)

// Configuration is used to define DataDog Tracer properties to be set when creating a new tracer
type Configuration struct {
	SampleRate          float64 // Sample rate for collecting spans (0-1)
	SpansBufferSize     int     // Buffer size for storing spans (default: 10000)
	DebugLoggingEnabled bool    // Enable or Disable DebugLogging (default: false)
	Enabled             bool    // Enable or Disable the tracer (default: false)
}

// NewTracer initializes the tracer object with a nil transport
func (c *Configuration) NewTracer() (opentracing.Tracer, io.Closer) {
	return c.NewTracerTransport(nil)
}

// NewTracerTransport initializes a new tracer with a supplied custom transport
func (c *Configuration) NewTracerTransport(transport tracer.Transport) (opentracing.Tracer, io.Closer) {
	var driver *tracer.Tracer
	if transport == nil {
		driver = tracer.NewTracer()
	} else {
		driver = tracer.NewTracerTransport(transport)
	}

	t := &Tracer{Tracer: driver}
	t.textPropagator = &textMapPropagator{t}

	t.Tracer.SetEnabled(c.Enabled)
	t.Tracer.DebugLoggingEnabled = c.DebugLoggingEnabled

	t.Tracer.SetSampleRate(c.SampleRate)
	if c.SpansBufferSize > 0 {
		t.Tracer.SetSpansBufferSize(c.SpansBufferSize)
	}

	// return the tracer as a closer so it can be cleaned up
	return t, t
}
