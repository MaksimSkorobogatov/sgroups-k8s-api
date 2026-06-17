package sockstats

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestStorageInterfaces(t *testing.T) {
	s := &Storage{}
	require.True(t, s.NamespaceScoped(), "subresource of namespaced host must be namespaced")
	require.Equal(t, []string{http.MethodGet}, s.ConnectMethods())

	obj, withSubpath, subpathPath := s.NewConnectOptions()
	require.Nil(t, obj, "NewConnectOptions should return nil — we parse query manually")
	require.False(t, withSubpath)
	require.Empty(t, subpathPath)

	require.NotNil(t, s.New())
	require.IsType(t, &v1alpha1.SocketStatList{}, s.New())
}

func TestParseOptionsWatchAndSelectors(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantWatch   bool
		wantFilters []v1alpha1.SocketStatSelector
		wantErr     string
	}{
		{
			name:  "empty",
			query: "",
		},
		{
			name:      "watch true",
			query:     "watch=true",
			wantWatch: true,
		},
		{
			name:    "watch invalid",
			query:   "watch=please",
			wantErr: `query parameter "watch"`,
		},
		{
			name:  "single selector",
			query: "selector=protocol=tcp,state=Listen,localPort=80",
			wantFilters: []v1alpha1.SocketStatSelector{
				{Protocol: "tcp", State: v1alpha1.ConnStateListen, LocalPort: 80},
			},
		},
		{
			name:  "multiple selectors OR-joined",
			query: "selector=localPort=80&selector=localPort=443",
			wantFilters: []v1alpha1.SocketStatSelector{
				{LocalPort: 80},
				{LocalPort: 443},
			},
		},
		{
			name:    "unknown selector key",
			query:   "selector=banana=yes",
			wantErr: `unknown selector key "banana"`,
		},
		{
			name:    "malformed selector entry",
			query:   "selector=localPort",
			wantErr: `selector entry must be "key=value"`,
		},
		{
			name:    "non-numeric localPort",
			query:   "selector=localPort=eighty",
			wantErr: `selector "localPort"`,
		},
		{
			name:  "family and pid",
			query: "selector=family=IPv4,pid=100,comm=sshd",
			wantFilters: []v1alpha1.SocketStatSelector{
				{Family: v1alpha1.IpAddrFamilyIPv4, PID: 100, Comm: "sshd"},
			},
		},
		{
			name:    "invalid state rejected (not silently coerced)",
			query:   "selector=state=asdasdasd",
			wantErr: `selector "state"`,
		},
		{
			name:    "lowercase state rejected (case-sensitive)",
			query:   "selector=state=established",
			wantErr: `selector "state"`,
		},
		{
			name:    "invalid family rejected",
			query:   "selector=family=banana",
			wantErr: `selector "family"`,
		},
		{
			name:  "valid state accepted",
			query: "selector=state=Established",
			wantFilters: []v1alpha1.SocketStatSelector{
				{State: v1alpha1.ConnStateEstablished},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sockstats?"+tt.query, nil)
			opts, err := parseOptions(req)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantWatch, opts.Watch)
			require.Equal(t, tt.wantFilters, opts.Selectors)
		})
	}
}

func TestSplitHelpers(t *testing.T) {
	require.Equal(t, []string{"a", "b", "c"}, splitCSV("a,b,c"))
	require.Equal(t, []string{"a"}, splitCSV("a"))
	require.Nil(t, splitCSV(""))

	k, v, ok := splitKV("foo=bar")
	require.True(t, ok)
	require.Equal(t, "foo", k)
	require.Equal(t, "bar", v)

	_, _, ok = splitKV("nokey")
	require.False(t, ok)

	// "=value" — empty key is technically valid syntactically; semantic
	// validation happens in assignSelectorField (rejected as unknown).
	k, v, ok = splitKV("=value")
	require.True(t, ok)
	require.Empty(t, k)
	require.Equal(t, "value", v)
}
