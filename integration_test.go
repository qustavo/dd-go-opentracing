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
	span := tr.StartSpan("child",
		opentracing.ChildOf(parent.Context()),
		opentracing.Tag{ServiceTagKey, "gochild"},
	)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return span
}

func TestIntegration(t *testing.T) {
	tr := NewTracer()
	tr.(*Tracer).DebugLoggingEnabled = true

	parent := tr.StartSpan("parent",
		opentracing.Tag{ServiceTagKey, "gotest"},
		opentracing.Tag{ResourceTagKey, "/user/{id}"},
	)
	parent.LogKV("foo", "bar", "ping", 0.546)

	spanChild(tr, parent, "child1").Finish()
	child := spanChild(tr, parent, "child2")
	parent.Finish()
	time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
	child.Finish()

	select {}

}
