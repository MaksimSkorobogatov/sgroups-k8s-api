package nft

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
	require.IsType(t, &v1alpha1.NftList{}, s.New())
}

func TestParseOptionsWatch(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantWatch bool
		wantErr   string
	}{
		{name: "empty", query: ""},
		{name: "watch true", query: "watch=true", wantWatch: true},
		{name: "watch false", query: "watch=false", wantWatch: false},
		{name: "watch invalid", query: "watch=please", wantErr: `query parameter "watch"`},
		// nft has no selectors; unrecognized params are simply ignored.
		{name: "unknown params ignored", query: "selector=protocol=tcp", wantWatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/nft?"+tt.query, nil)
			opts, err := parseOptions(req)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantWatch, opts.Watch)
		})
	}
}
