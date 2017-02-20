//+build integration

package ddtracer

import (
	"math/rand"
	"testing"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
)

func spanChild(tr opentracing.Tracer, parent opentracing.Span, op string) opentracing.Span {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	child := tr.StartSpan("child", opentracing.ChildOf(parent.Context()))
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return child
}

func TestIntegration(t *testing.T) {
	tr := NewTracer()
	tr.(*Tracer).DebugLoggingEnabled = true

	parent := tr.StartSpan("parent")
	parent.LogKV("foo", "bar", "ping", 0.546)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	spanChild(tr, parent, "child1").Finish()
	spanChild(tr, parent, "child2").Finish()
	parent.Finish()

	select {}

}
