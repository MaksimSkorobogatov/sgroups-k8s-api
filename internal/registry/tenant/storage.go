package tenant

import (
	"sgroups.io/sgroups-k8s-api/internal/registry/generic"
	"sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

// NewStorage creates a Tenant storage backed by gRPC.
func NewStorage(c *client.Client, opts options.StorageOptions) *generic.Storage[*v1alpha1.Tenant, *v1alpha1.TenantList] {
	return generic.NewStorage(
		&backend{client: c},
		opts,
		&tenantStrategy{},
		func() *v1alpha1.Tenant { return &v1alpha1.Tenant{} },
		func() *v1alpha1.TenantList { return &v1alpha1.TenantList{} },
		"tenant",
	)
}
