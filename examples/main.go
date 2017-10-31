package main

import (
	"errors"
	"math/rand"
	"time"

	dd "github.com/gchaincl/dd-go-opentracing"
	opentracing "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

func spanChild(tr opentracing.Tracer, parent opentracing.Span, service, op string) opentracing.Span {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	span := tr.StartSpan(op, opentracing.ChildOf(parent.Context()))
	ext.PeerService.Set(span, service)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	return span
}

func main() {
	tr := dd.NewTracer()

	for {
		// Start the parent Span
		parent := tr.StartSpan("pylons.request",
			opentracing.Tag{Key: "foo", Value: "bar"},
			opentracing.Tag{Key: "ping", Value: 0.546},
		)
		// Set Service name and Resource
		ext.PeerService.Set(parent, "pylons")
		ext.Component.Set(parent, "/users/{id}")

		// Set env
		dd.EnvTag.Set(parent, "test")

		spanChild(tr, parent, "redis", "redis.command").Finish()
		async := spanChild(tr, parent, "queue", "async.job")
		parent.Finish()
		time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
		async.LogFields(
			log.Error(errors.New("boom")),
		)
		async.Finish()
	}
}
