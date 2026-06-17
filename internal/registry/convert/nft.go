package convert

import (
	"encoding/json"

	agentv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/agent/v1"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// NftListFromProto flattens hosts[0].nft into a K8s NftList; extra hosts are
// ignored since the subresource queries one host by URL.
//
//nolint:dupl // mirrors SocketStatListFromProto by design.
func NftListFromProto(in []*sgroupsv1.HostResp_Nft_Host) *v1alpha1.NftList {
	out := &v1alpha1.NftList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.KindNftList,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Items: []v1alpha1.Nft{},
	}
	if len(in) == 0 {
		return out
	}
	first := in[0]
	if first == nil {
		return out
	}
	out.Host = v1alpha1.ResourceIdentifier{
		Name:      first.GetName(),
		Namespace: first.GetNamespace(),
	}
	nfts := first.GetNft()
	out.Items = make([]v1alpha1.Nft, 0, len(nfts))
	for _, n := range nfts {
		if n == nil {
			continue
		}
		out.Items = append(out.Items, nftFromProto(n))
	}

	return out
}

// NftListRequest builds the gRPC List request for a single host (no filters).
func NftListRequest(namespace, name string) *sgroupsv1.HostReq_Nft_List {
	return &sgroupsv1.HostReq_Nft_List{
		Selectors: []*sgroupsv1.HostReq_Nft_FieldSelector{
			{Name: name, Namespace: namespace},
		},
	}
}

// NftWatchRequest mirrors NftListRequest for the Watch RPC.
func NftWatchRequest(namespace, name string) *sgroupsv1.HostReq_Nft_Watch {
	return &sgroupsv1.HostReq_Nft_Watch{
		Selectors: []*sgroupsv1.HostReq_Nft_FieldSelector{
			{Name: name, Namespace: namespace},
		},
	}
}

// nftFromProto converts agentv1.Nftables → v1alpha1.Nft, embedding `json` as
// structured JSON only when valid so a malformed payload can't break the response.
func nftFromProto(in *agentv1.Nftables) v1alpha1.Nft {
	if in == nil {
		return v1alpha1.Nft{}
	}
	out := v1alpha1.Nft{Text: in.GetText()}
	if raw := in.GetJson(); raw != "" && json.Valid([]byte(raw)) {
		out.JSON = &runtime.RawExtension{Raw: []byte(raw)}
	}

	return out
}
