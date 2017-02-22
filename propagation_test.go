package ddtracer

import (
	"net/http"
	"testing"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropagationInject(t *testing.T) {
	tr := NewTracer()
	span := tr.StartSpan("span").(*Span)
	span.SpanID = 0xaa
	span.TraceID = 0xbb

	req, _ := http.NewRequest("GET", "/", nil)
	err := tr.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	require.NoError(t, err)

	assert.Equal(t, "aa", req.Header.Get("Dd-Trace-Spanid"))
	assert.Equal(t, "bb", req.Header.Get("Dd-Trace-Traceid"))

	t.Run("Parent doesn't set ParentID", func(t *testing.T) {
		assert.Empty(t, req.Header.Get("Dd-Trace-Parentid"))
	})

	t.Run("Child sets ParentID", func(t *testing.T) {
		span.ParentID = 0xcc
		err := tr.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
		require.NoError(t, err)
		assert.Equal(t, "cc", req.Header.Get("Dd-Trace-Parentid"))

	})
}

func TestPropagationExtract(t *testing.T) {
	tr := NewTracer()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Dd-Trace-Spanid", "aa")
	req.Header.Set("Dd-Trace-Traceid", "bb")
	req.Header.Set("Dd-Trace-Parentid", "cc")

	spanContext, err := tr.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	require.NoError(t, err)

	ctx := spanContext.(*SpanContext)
	span, ok := tracer.SpanFromContext(ctx.ctx)
	require.True(t, ok)

	assert.Equal(t, uint64(0xaa), span.SpanID)
	assert.Equal(t, uint64(0xbb), span.TraceID)
	assert.Equal(t, uint64(0xcc), span.ParentID)

	t.Run("CorruptedContext", func(t *testing.T) {
		for _, key := range []string{"Spanid", "Parentid", "Traceid"} {
			_, err := tr.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(http.Header{
				"Dd-Trace-" + key: []string{"NaN"},
			}))
			assert.Equal(t, opentracing.ErrSpanContextCorrupted, err)
		}
	})

}
