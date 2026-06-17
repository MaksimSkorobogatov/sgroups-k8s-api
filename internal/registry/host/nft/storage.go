// Package nft implements the read-only hosts/nft subresource: proxies
// SGroupsHostsAPI.{List,Watch}Nft to the client and surfaces the host's
// nftables ruleset; ?watch=true toggles list vs a chunked JSON stream.
package nft

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"sgroups.io/sgroups-k8s-api/internal/registry/convert"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

// Storage serves hosts/nft: GET for list, GET ?watch=true for stream.
type Storage struct {
	client *client.Client
}

func NewStorage(c *client.Client, _ registryoptions.StorageOptions) *Storage {
	return &Storage{client: c}
}

// Compile-time guards for the K8s endpoint installer.
var (
	_ rest.Storage   = (*Storage)(nil)
	_ rest.Scoper    = (*Storage)(nil)
	_ rest.Connecter = (*Storage)(nil)
)

func (s *Storage) NamespaceScoped() bool    { return true }
func (s *Storage) New() runtime.Object      { return &v1alpha1.NftList{} }
func (s *Storage) Destroy()                 {}
func (s *Storage) ConnectMethods() []string { return []string{http.MethodGet} }

// NewConnectOptions returns nil so query parameters are parsed manually inside
// Connect, avoiding scheme registration of an options type.
func (s *Storage) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

// Connect returns the per-request handler. id is the host name from the URL;
// namespace comes from the request context populated by the installer.
func (s *Storage) Connect(ctx context.Context, id string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, _ := request.NamespaceFrom(ctx)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		opts, err := parseOptions(req)
		if err != nil {
			responder.Error(badRequest(err))

			return
		}
		if opts.Watch {
			s.serveWatch(req.Context(), w, namespace, id)

			return
		}
		s.serveList(req.Context(), namespace, id, responder)
	}), nil
}

// serveList performs one ListNft RPC; responder handles content negotiation,
// status code, and writing.
func (s *Storage) serveList(ctx context.Context, namespace, name string, responder rest.Responder) {
	resp, err := s.client.Hosts.ListNft(ctx, convert.NftListRequest(namespace, name))
	if err != nil {
		responder.Error(err)

		return
	}
	responder.Object(http.StatusOK, convert.NftListFromProto(resp.GetHosts()))
}

// serveWatch proxies WatchNft into a chunked JSON stream; each upstream event
// becomes one JSON-encoded NftList object.
func (s *Storage) serveWatch(ctx context.Context, w http.ResponseWriter, namespace, name string) {
	stream, err := s.client.Hosts.WatchNft(ctx, convert.NftWatchRequest(namespace, name))
	if err != nil {
		writeStreamError(w, err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		evt, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			// Mid-stream failure: headers already flushed → emit a Status
			// envelope in the body and stop.
			_ = enc.Encode(map[string]string{"kind": "Status", "status": "Failure", "message": err.Error()})
			if flusher != nil {
				flusher.Flush()
			}

			return
		}
		if encErr := enc.Encode(convert.NftListFromProto(evt.GetHosts())); encErr != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// parseOptions reads ?watch=bool — nft has no selectors.
func parseOptions(req *http.Request) (*v1alpha1.NftOptions, error) {
	opts := &v1alpha1.NftOptions{}
	if v := req.URL.Query().Get("watch"); v != "" {
		watch, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.New(`query parameter "watch" must be a boolean ("true"/"false")`)
		}
		opts.Watch = watch
	}

	return opts, nil
}

func badRequest(err error) error { return apierrors.NewBadRequest(err.Error()) }

// writeStreamError emits HTTP 500 + a Status envelope when the watch stream
// fails before any bytes were flushed.
func writeStreamError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"kind":    "Status",
		"status":  "Failure",
		"message": err.Error(),
	})
}
