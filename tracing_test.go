package ddtracer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/DataDog/dd-trace-go/tracer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type env struct {
	reqs []*http.Request
	ts   *httptest.Server
	tr   opentracing.Tracer
}

func newEnv() *env {
	e := &env{}
	e.ts = httptest.NewServer(e)
	url, _ := url.Parse(e.ts.URL)
	hostPort := strings.Split(url.Host, ":")
	e.tr = NewTracerTransport(
		//		tracer.NewTransport("localhost", "8000"),
		tracer.NewTransport(hostPort[0], hostPort[1]),
	)
	return e
}

func (e *env) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.URL.String(), "v0.3") {
		w.WriteHeader(415)
		return
	}

	buf, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Fatalln(err)
	}

	newReq, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buf)))
	if err != nil {
		log.Fatalln(err)
	}

	e.reqs = append(e.reqs, newReq)
}

func (e *env) reset() {
	e.reqs = make([]*http.Request, 0)
}

func (e *env) close() {
	e.reset()
	e.ts.Close()
}

func TestSpansParenthood(t *testing.T) {
	env := newEnv()
	defer env.close()

	pSpan := env.tr.StartSpan("parent")
	cSpan := env.tr.StartSpan("child", opentracing.ChildOf(pSpan.Context()))
	cSpan.Finish()
	pSpan.Finish()

	err := env.tr.(*Tracer).FlushTraces()
	require.NoError(t, err)

	var spans [][]*tracer.Span
	if err := json.NewDecoder(env.reqs[0].Body).Decode(&spans); err != nil {
		require.NoError(t, err)
	}

	child := spans[0][0]
	parent := spans[0][1]

	assert.Equal(t, "child", child.Name)
	assert.Equal(t, parent.SpanID, child.ParentID)
	assert.Equal(t, child.TraceID, parent.TraceID)
}

func TestSpanTags(t *testing.T) {
	span := NewTracer().StartSpan("test")
	span.LogKV(
		"foo", "bar",
		"key", "val",
		"int", 123,
		"metric", 0.1,
	)

	assert.Equal(t, "bar", span.(*Span).GetMeta("foo"))
	assert.Equal(t, "val", span.(*Span).GetMeta("key"))
	assert.Equal(t, "123", span.(*Span).GetMeta("int"))
	assert.Equal(t, 0.1, span.(*Span).Metrics["metric"])
}

func TestDDParams(t *testing.T) {
	span := NewTracer().StartSpan("test",
		opentracing.Tag{ServiceTagKey, "/bin/laden"},
		opentracing.Tag{ResourceTagKey, "/user/{id}"},
		opentracing.Tag{"user_agent", "firefox"},
	).(*Span)

	assert.Equal(t, "/bin/laden", span.Service)
	assert.Equal(t, "/user/{id}", span.Resource)
	assert.Equal(t, "firefox", span.GetMeta("user_agent"))
}

func TestSpanSetOperationName(t *testing.T) {
	span := NewTracer().
		StartSpan("test").
		SetOperationName("op")
	assert.Equal(t, "op", span.(*Span).Name)
}
