package convert

import (
	"encoding/json"
	"testing"

	agentv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/agent/v1"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	"github.com/stretchr/testify/require"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestNftListFromProto(t *testing.T) {
	in := []*sgroupsv1.HostResp_Nft_Host{
		{
			Name:      "host-1",
			Namespace: "default",
			Nft: []*agentv1.Nftables{
				{
					Text: "table inet filter {\n}\n",
					Json: `{"nftables":[{"table":{"family":"inet","name":"filter"}}]}`,
				},
				nil,                           // must be skipped
				{Text: "table ip nat {\n}\n"}, // no json
			},
		},
	}

	out := NftListFromProto(in)
	require.NotNil(t, out)
	require.Equal(t, v1alpha1.KindNftList, out.Kind)
	require.Equal(t, v1alpha1.SchemeGroupVersion.String(), out.APIVersion)
	require.Equal(t, "host-1", out.Host.Name)
	require.Equal(t, "default", out.Host.Namespace)
	require.Len(t, out.Items, 2)

	require.Equal(t, "table inet filter {\n}\n", out.Items[0].Text)
	require.NotNil(t, out.Items[0].JSON, "valid proto json must be embedded")
	require.JSONEq(t,
		`{"nftables":[{"table":{"family":"inet","name":"filter"}}]}`,
		string(out.Items[0].JSON.Raw),
		"valid proto json must be embedded verbatim as structured JSON",
	)

	require.Equal(t, "table ip nat {\n}\n", out.Items[1].Text)
	require.Nil(t, out.Items[1].JSON, "absent json yields a nil pointer (field omitted)")
}

func TestNftFromProtoInvalidJSON(t *testing.T) {
	// Defensive: a malformed json payload must NOT be embedded, otherwise it
	// would corrupt the entire serialized response. Text is still surfaced.
	got := nftFromProto(&agentv1.Nftables{Text: "ruleset", Json: "{not valid json"})
	require.Equal(t, "ruleset", got.Text)
	require.Nil(t, got.JSON, "invalid json must be dropped, not embedded")
}

// TestNftMarshalOmitsEmptyJSON pins the empty-value contract: a text-only entry
// must NOT emit "json": null — the field is omitted entirely.
func TestNftMarshalOmitsEmptyJSON(t *testing.T) {
	list := NftListFromProto([]*sgroupsv1.HostResp_Nft_Host{{
		Name: "h", Namespace: "ns",
		Nft: []*agentv1.Nftables{
			{Text: "table ip nat {}"},            // text only
			{Text: "t", Json: `{"nftables":[]}`}, // text + json
		},
	}})
	b, err := json.Marshal(list)
	require.NoError(t, err)
	require.NotContains(t, string(b), `"json":null`, "empty json must be omitted, not serialized as null")
	require.Contains(t, string(b), `"json":{"nftables":[]}`, "present json must embed inline")
}

func TestNftFromProtoNil(t *testing.T) {
	require.Equal(t, v1alpha1.Nft{}, nftFromProto(nil))
}

func TestNftListFromProtoEmpty(t *testing.T) {
	out := NftListFromProto(nil)
	require.NotNil(t, out)
	require.Empty(t, out.Items)
	require.Empty(t, out.Host.Name)
	require.Empty(t, out.Host.Namespace)
}

func TestNftListFromProtoSingleEntry(t *testing.T) {
	// The subresource always queries one host; anything beyond hosts[0] is
	// silently ignored (server-bug defence), mirroring socket statistics.
	in := []*sgroupsv1.HostResp_Nft_Host{
		{Name: "expected", Nft: []*agentv1.Nftables{{Text: "a"}}},
		{Name: "unexpected-extra", Nft: []*agentv1.Nftables{{Text: "b"}}},
	}

	out := NftListFromProto(in)
	require.Equal(t, "expected", out.Host.Name, "host attribution comes from hosts[0]")
	require.Len(t, out.Items, 1, "only the first host's nft should appear")
	require.Equal(t, "a", out.Items[0].Text)
}

func TestNftListFromProtoNilFirst(t *testing.T) {
	out := NftListFromProto([]*sgroupsv1.HostResp_Nft_Host{nil})
	require.NotNil(t, out)
	require.Empty(t, out.Items)
}

func TestNftListRequest(t *testing.T) {
	req := NftListRequest("default", "host-1")
	require.NotNil(t, req)
	require.Len(t, req.Selectors, 1)
	require.Equal(t, "host-1", req.Selectors[0].Name)
	require.Equal(t, "default", req.Selectors[0].Namespace)
}

func TestNftWatchRequest(t *testing.T) {
	req := NftWatchRequest("ns", "h")
	require.NotNil(t, req)
	require.Len(t, req.Selectors, 1)
	require.Equal(t, "h", req.Selectors[0].Name)
	require.Equal(t, "ns", req.Selectors[0].Namespace)
}

// Sanity check that compile-time we still wire proto types correctly.
var (
	_ = sgroupsv1.HostReq_Nft_List{}
	_ = sgroupsv1.HostResp_Nft_Watch{}
)
