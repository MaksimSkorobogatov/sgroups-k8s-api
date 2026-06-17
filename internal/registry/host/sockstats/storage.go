// Package sockstats implements the read-only hosts/sockstats subresource:
// proxies SGroupsHostsAPI.{List,Watch}SocketStatistics to a chunked JSON
// stream; ?watch=true toggles list vs streaming.
package sockstats

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

// Storage serves hosts/sockstats: GET for list, GET ?watch=true for stream.
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
func (s *Storage) New() runtime.Object      { return &v1alpha1.SocketStatList{} }
func (s *Storage) Destroy()                 {}
func (s *Storage) ConnectMethods() []string { return []string{http.MethodGet} }

// NewConnectOptions returns nil so query parameters are parsed manually
// inside Connect, avoiding scheme registration of an options type.
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
			s.serveWatch(req.Context(), w, namespace, id, opts.Selectors)

			return
		}
		s.serveList(req.Context(), namespace, id, opts.Selectors, responder)
	}), nil
}

// serveList performs one ListSocketStatistics RPC; responder handles content
// negotiation, status code, and writing.
func (s *Storage) serveList(
	ctx context.Context,
	namespace, name string,
	selectors []v1alpha1.SocketStatSelector,
	responder rest.Responder,
) {
	req := convert.SocketStatListRequest(namespace, name, selectors)
	resp, err := s.client.Hosts.ListSocketStatistics(ctx, req)
	if err != nil {
		responder.Error(err)

		return
	}
	responder.Object(http.StatusOK, convert.SocketStatListFromProto(resp.GetHosts()))
}

// serveWatch proxies WatchSocketStatistics into a chunked JSON stream; each
// upstream event becomes one JSON-encoded SocketStatList object.
func (s *Storage) serveWatch(ctx context.Context, w http.ResponseWriter, namespace, name string, selectors []v1alpha1.SocketStatSelector) {
	stream, err := s.client.Hosts.WatchSocketStatistics(ctx, convert.SocketStatWatchRequest(namespace, name, selectors))
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
		if encErr := enc.Encode(convert.SocketStatListFromProto(evt.GetHosts())); encErr != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// parseOptions reads ?watch=bool and zero-or-more ?selector=key=val,key=val
// entries. Multiple selector params are OR-joined.
func parseOptions(req *http.Request) (*v1alpha1.SocketStatOptions, error) {
	q := req.URL.Query()
	opts := &v1alpha1.SocketStatOptions{}
	if v := q.Get("watch"); v != "" {
		w, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.New(`query parameter "watch" must be a boolean ("true"/"false")`)
		}
		opts.Watch = w
	}
	for _, sel := range q["selector"] {
		parsed, err := parseSelector(sel)
		if err != nil {
			return nil, err
		}
		opts.Selectors = append(opts.Selectors, parsed)
	}

	return opts, nil
}

// parseSelector turns "protocol=tcp,state=Listen,localPort=80" into a
// SocketStatSelector. Unknown keys are rejected so typos fail loudly.
func parseSelector(raw string) (v1alpha1.SocketStatSelector, error) {
	sel := v1alpha1.SocketStatSelector{}
	if raw == "" {
		return sel, nil
	}
	for _, part := range splitCSV(raw) {
		k, v, ok := splitKV(part)
		if !ok {
			return sel, errors.New(`selector entry must be "key=value", got: ` + part)
		}
		if err := assignSelectorField(&sel, k, v); err != nil {
			return sel, err
		}
	}

	return sel, nil
}

// selectorSetter assigns one parsed string to a typed field on the selector.
type selectorSetter func(sel *v1alpha1.SocketStatSelector, value string) error

var selectorSetters = map[string]selectorSetter{
	"protocol":   setString(func(s *v1alpha1.SocketStatSelector) *string { return &s.Protocol }),
	"localAddr":  setString(func(s *v1alpha1.SocketStatSelector) *string { return &s.LocalAddr }),
	"remoteAddr": setString(func(s *v1alpha1.SocketStatSelector) *string { return &s.RemoteAddr }),
	"ifname":     setString(func(s *v1alpha1.SocketStatSelector) *string { return &s.Ifname }),
	"comm":       setString(func(s *v1alpha1.SocketStatSelector) *string { return &s.Comm }),
	"family":     setFamily,
	"state":      setConnState,
	"localPort":  setInt32(func(s *v1alpha1.SocketStatSelector) *int32 { return &s.LocalPort }, "localPort"),
	"remotePort": setInt32(func(s *v1alpha1.SocketStatSelector) *int32 { return &s.RemotePort }, "remotePort"),
	"pid":        setInt32(func(s *v1alpha1.SocketStatSelector) *int32 { return &s.PID }, "pid"),
	"inode":      setInode,
}

// setString builds a setter for any string field via a field-address closure.
func setString(fieldPtr func(*v1alpha1.SocketStatSelector) *string) selectorSetter {
	return func(s *v1alpha1.SocketStatSelector, v string) error {
		*fieldPtr(s) = v

		return nil
	}
}

// setInode parses int64 — the only non-int32 numeric field on the selector.
func setInode(s *v1alpha1.SocketStatSelector, v string) error {
	inode, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return errors.New(`selector "inode" must be an integer`)
	}
	s.Inode = inode

	return nil
}

// setConnState validates state — unknown values are rejected, not silently dropped.
func setConnState(s *v1alpha1.SocketStatSelector, v string) error {
	st := v1alpha1.ConnState(v)
	if err := st.Validate(); err != nil {
		return errors.New(`selector "state": ` + err.Error())
	}
	s.State = st

	return nil
}

// setFamily validates family — unknown values are rejected, not silently dropped.
func setFamily(s *v1alpha1.SocketStatSelector, v string) error {
	f := v1alpha1.IpAddrFamily(v)
	if err := f.Validate(); err != nil {
		return errors.New(`selector "family": ` + err.Error())
	}
	s.Family = f

	return nil
}

func assignSelectorField(sel *v1alpha1.SocketStatSelector, key, value string) error {
	set, ok := selectorSetters[key]
	if !ok {
		return errors.New(`unknown selector key "` + key + `"`)
	}

	return set(sel, value)
}

// setInt32 builds a setter for any int32 field via a field-address closure.
// `name` is used only for the error message.
func setInt32(fieldPtr func(*v1alpha1.SocketStatSelector) *int32, name string) selectorSetter {
	return func(s *v1alpha1.SocketStatSelector, v string) error {
		port, err := parseInt32(v)
		if err != nil {
			return errors.New(`selector "` + name + `": ` + err.Error())
		}
		*fieldPtr(s) = port

		return nil
	}
}

func parseInt32(s string) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, errors.New(`must be an integer in [-2147483648, 2147483647]`)
	}

	return int32(v), nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0, 4)
	start := 0
	for i, r := range s {
		if r == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}

	return append(out, s[start:])
}

func splitKV(s string) (key, value string, ok bool) {
	for i, r := range s {
		if r == '=' {
			return s[:i], s[i+1:], true
		}
	}

	return s, "", false
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
