package apiserver

import (
	"k8s.io/apiserver/pkg/registry/rest"

	"sgroups.io/sgroups-k8s-api/internal/registry/host/nft"
	"sgroups.io/sgroups-k8s-api/internal/registry/host/sockstats"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

// installSubresources adds entries with "<resource>/<subresource>" keys to
// the generated storage map; the K8s installer maps them to
// /apis/<group>/<version>/namespaces/{ns}/<resource>/{name}/<subresource>.
func installSubresources(m map[string]map[string]rest.Storage, c *client.Client, opts registryoptions.StorageOptions) {
	v1alpha1Map, ok := m[v1alpha1.SchemeGroupVersion.Version]
	if !ok {
		return
	}
	v1alpha1Map["hosts/"+v1alpha1.SubresourceHostSocketStats] = sockstats.NewStorage(c, opts)
	v1alpha1Map["hosts/"+v1alpha1.SubresourceHostNft] = nft.NewStorage(c, opts)
}
