package sockstats

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commonpb "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"

	"sgroups.io/sgroups-k8s-api/internal/backend"
	"sgroups.io/sgroups-k8s-api/internal/mock"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/internal/registry/testutil"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// fakeResponder captures Responder.Object/Error so tests can assert without
// an HTTP-level codec stack.
type fakeResponder struct {
	statusCode int
	object     runtime.Object
	err        error
}

func (f *fakeResponder) Object(code int, obj runtime.Object) {
	f.statusCode = code
	f.object = obj
}

func (f *fakeResponder) Error(err error) { f.err = err }

// seedHost upserts a host into the mock so ListSocketStatistics returns stats.
func seedHost(t *testing.T, mb *mock.MockBackend, namespace, name string) {
	t.Helper()
	_, err := mb.UpsertHosts(context.Background(), &sgroupsv1.HostReq_Upsert{
		Hosts: []*sgroupsv1.Host{
			{
				Metadata: &commonpb.Metadata{Name: name, Namespace: namespace},
				Spec:     &sgroupsv1.Host_Spec{DisplayName: name},
			},
		},
	})
	require.NoError(t, err)
}

func TestStorage_ListSocketStats_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "default", "host-1")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "host-1", nil, resp)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/sockstats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NoError(t, resp.err, "list path should not surface an error")
	require.Equal(t, http.StatusOK, resp.statusCode)

	list, ok := resp.object.(*v1alpha1.SocketStatList)
	require.True(t, ok, "Responder.Object must receive *SocketStatList, got %T", resp.object)
	require.NotEmpty(t, list.Items, "mock seeds at least one fabricated SockStat for an existing host")
	for _, item := range list.Items {
		require.Equal(t, "tcp", item.Protocol)
	}
}

func TestStorage_ListSocketStats_FilterByState_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "prod", "web-01")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "prod")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "web-01", nil, resp)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/sockstats?selector=state=Established", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NoError(t, resp.err)
	list := resp.object.(*v1alpha1.SocketStatList)
	for _, item := range list.Items {
		require.Equal(t, v1alpha1.ConnStateEstablished, item.State,
			"selector state=Established must filter out LISTEN entries")
	}
}

func TestStorage_WatchSocketStats_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "default", "host-w")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	// httptest.ResponseRecorder is not concurrent-safe; use io.Pipe so the
	// handler writes and the test reads without a shared buffer.
	pr, pw := io.Pipe()
	defer pr.Close()

	parent, cancel := context.WithCancel(context.Background())
	ctx := request.WithNamespace(parent, "default")

	handler, err := s.Connect(ctx, "host-w", nil, &fakeResponder{})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/sockstats?watch=true", nil).WithContext(ctx)

	done := make(chan struct{})
	rw := &pipeResponseWriter{w: pw, header: http.Header{}}
	go func() {
		handler.ServeHTTP(rw, req)
		_ = pw.Close()
		close(done)
	}()

	// Decoder blocks until the handler writes the first chunk; mock emits
	// exactly one snapshot, so this returns immediately.
	dec := json.NewDecoder(pr)
	var list v1alpha1.SocketStatList
	require.NoError(t, dec.Decode(&list))
	require.Equal(t, v1alpha1.KindSocketStatList, list.Kind)
	require.NotEmpty(t, list.Items)

	// Cancel ctx so handler unblocks from stream.Recv(), then await goroutine
	// exit so the race detector sees a clean boundary.
	cancel()
	_, _ = io.Copy(io.Discard, pr) // drain anything else that arrived
	<-done

	require.Equal(t, http.StatusOK, rw.status)
	require.Equal(t, "application/json", rw.header.Get("Content-Type"))
}

// pipeResponseWriter is a no-mutex http.ResponseWriter backed by an io.Writer
// (handler writes serially from its own goroutine).
type pipeResponseWriter struct {
	w      io.Writer
	header http.Header
	status int
}

func (p *pipeResponseWriter) Header() http.Header         { return p.header }
func (p *pipeResponseWriter) Write(b []byte) (int, error) { return p.w.Write(b) }
func (p *pipeResponseWriter) WriteHeader(status int)      { p.status = status }

// Flush satisfies http.Flusher so serveWatch takes the flushing branch.
func (p *pipeResponseWriter) Flush() {}

func TestStorage_BadQueryParam_E2E(t *testing.T) {
	mb := mock.New()
	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "ignored", nil, resp)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/sockstats?selector=banana=yes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Error(t, resp.err, "unknown selector key must surface as Bad Request")
}

// oversizedSelectors builds a ~5 MiB query string to push the gRPC request
// past the default server-side 4 MiB MaxRecvMsgSize.
func oversizedSelectors(t *testing.T) string {
	t.Helper()
	const (
		entries = 5000
		pad     = 1024
	)
	padding := strings.Repeat("x", pad)
	var b strings.Builder
	for i := range entries {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString("selector=comm=")
		b.WriteString(padding)
	}

	return b.String()
}

func TestStorage_ListSocketStats_OversizedRequest_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "default", "host-1")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "host-1", nil, resp)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/sockstats?"+oversizedSelectors(t), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// 5 MiB request vs 4 MiB server cap → ResourceExhausted, surfaced via Error.
	require.Error(t, resp.err)
	require.Contains(t, strings.ToLower(resp.err.Error()), "resourceexhausted")
}

func TestStorage_WatchSocketStats_OversizedRequest_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "default", "host-w")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "host-w", nil, resp)
	require.NoError(t, err)

	pr, pw := io.Pipe()
	defer pr.Close()
	rw := &pipeResponseWriter{w: pw, header: http.Header{}}

	req := httptest.NewRequest(http.MethodGet, "/sockstats?watch=true&"+oversizedSelectors(t), nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rw, req)
		_ = pw.Close()
		close(done)
	}()

	body, err := io.ReadAll(pr)
	require.NoError(t, err, "reading drained pipe")
	<-done

	// gRPC server-streaming sends the request lazily: error arrives on the
	// first Recv() after WriteHeader(200) has flushed — so the handler must
	// embed a Status envelope in the body instead of setting 5xx.
	require.Equal(t, http.StatusOK, rw.status)
	require.Contains(t, strings.ToLower(string(body)), "resourceexhausted")
	require.Contains(t, string(body), `"status":"Failure"`)
}
