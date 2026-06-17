package nft

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

// fakeResponder captures Responder.Object/Error so tests can assert without an
// HTTP-level codec stack.
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

// seedHost upserts a host into the mock so ListNft returns a ruleset.
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

func TestStorage_ListNft_E2E(t *testing.T) {
	mb := mock.New()
	seedHost(t, mb, "default", "host-1")

	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "host-1", nil, resp)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/nft", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.NoError(t, resp.err, "list path should not surface an error")
	require.Equal(t, http.StatusOK, resp.statusCode)

	list, ok := resp.object.(*v1alpha1.NftList)
	require.True(t, ok, "Responder.Object must receive *NftList, got %T", resp.object)
	require.Equal(t, "host-1", list.Host.Name)
	require.Equal(t, "default", list.Host.Namespace)
	require.NotEmpty(t, list.Items, "mock seeds a fabricated ruleset for an existing host")
	require.NotEmpty(t, list.Items[0].Text, "text ruleset must be surfaced")
	require.NotNil(t, list.Items[0].JSON, "mock seeds a json ruleset")
	require.True(t, json.Valid(list.Items[0].JSON.Raw),
		"structured json must survive the round-trip as valid JSON")
}

func TestStorage_ListNft_UnknownHost_E2E(t *testing.T) {
	// No host seeded → backend drops the selector → empty response. The
	// subresource must return an empty NftList rather than erroring.
	mb := mock.New()
	c, cleanup := testutil.NewBufconnClient(t, backend.Backend{Hosts: mb})
	defer cleanup()

	s := NewStorage(c, registryoptions.StorageOptions{})

	ctx := request.WithNamespace(context.Background(), "default")
	resp := &fakeResponder{}
	handler, err := s.Connect(ctx, "ghost", nil, resp)
	require.NoError(t, err)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nft", nil))

	require.NoError(t, resp.err)
	list := resp.object.(*v1alpha1.NftList)
	require.Empty(t, list.Items)
}

func TestStorage_WatchNft_E2E(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/nft?watch=true", nil).WithContext(ctx)

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
	var list v1alpha1.NftList
	require.NoError(t, dec.Decode(&list))
	require.Equal(t, v1alpha1.KindNftList, list.Kind)
	require.NotEmpty(t, list.Items)
	require.NotNil(t, list.Items[0].JSON)
	require.True(t, json.Valid(list.Items[0].JSON.Raw))

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

	req := httptest.NewRequest(http.MethodGet, "/nft?watch=please", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Error(t, resp.err, "invalid watch value must surface as Bad Request")
}
