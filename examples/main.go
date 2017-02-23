package main

import (
	"log"
	"math/rand"
	"time"

	dd "github.com/gchaincl/dd-go-opentracing"
	opentracing "github.com/opentracing/opentracing-go"
)

func spanChild(tr opentracing.Tracer, parent opentracing.Span, op string) opentracing.Span {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	span := tr.StartSpan("child",
		opentracing.ChildOf(parent.Context()),
		opentracing.Tag{dd.ServiceTagKey, "gochild"},
	)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return span
}

func main() {
	tr := dd.NewTracer()
	tr.(*dd.Tracer).DebugLoggingEnabled = true

	parent := tr.StartSpan("parent",
		opentracing.Tag{dd.ServiceTagKey, "gotest"},
		opentracing.Tag{dd.ResourceTagKey, "/user/{id}"},
	)
	parent.LogKV("foo", "bar", "ping", 0.546)

	spanChild(tr, parent, "child1").Finish()
	child := spanChild(tr, parent, "child2")
	parent.Finish()
	time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
	child.Finish()

	if err := tr.(*dd.Tracer).FlushTraces(); err != nil {
		log.Fatalln(err)
	}
}
