// Package ddtracer is a DataDog's tracer (https://github.com/DataDog/dd-trace-go) wrapper for the OpenTracing API.
// The goal of the wrapper is to exploit all the functionalities provided by DataDog witout leaving the OpenTracing API nor having to deal with reflecion/type casting.
// Although both API have similar semantics, there's some concepts in DataDog which doesn't fit with OpenTracing specs.
//
// Service and Resource has been implemented through opentracing-go/ext.{PeerService,Component},
// to invoke Span.SetError, opentracing-go/log.Error() has been used (see examples below).
//
// This is an OpenTracing API wrapper, all the methods implementing the specifications are described in detail on official documentation.
package ddtracer
