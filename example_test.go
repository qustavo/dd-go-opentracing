package ddtracer

import (
	"fmt"
	"math/rand"
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

func ExampleNewTracer() {
	t := NewTracer()

	// To access DataDog tracer's you will need to type cast

	// Let's enable Debug Log
	t.(*Tracer).DebugLoggingEnabled = true
	// And flush all the remaining traces
	t.(*Tracer).FlushTraces()
}

func ExampleTracer_StartSpan() {
	span := NewTracer().StartSpan("span")

	// Let set DataDog's specific attrs Service and Resource
	span.LogKV(
		ServiceTagKey, "gotest",
		ResourceTagKey, "/user/{id}",
	)

	// To Set metrics, we need to pass a float64 value type
	span.LogKV("elapsed", 0.1234)

	// Everything else, is going to be treated as Meta
	span.LogKV("query", "SELECT data FROM dogs")

	span.Finish()
	ddspan := span.(*Span)
	fmt.Println("service  =", ddspan.Service)
	fmt.Println("resource =", ddspan.Resource)
	fmt.Println("elapsed  =", ddspan.Metrics["elapsed"])
	fmt.Println("query    =", ddspan.GetMeta("query"))
	// Output:
	// service  = gotest
	// resource = /user/{id}
	// elapsed  = 0.1234
	// query    = SELECT data FROM dogs
}
