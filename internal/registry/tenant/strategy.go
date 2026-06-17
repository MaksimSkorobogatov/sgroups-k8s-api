package tenant

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

type tenantStrategy struct{}

func (s *tenantStrategy) ValidateCreate(_ context.Context, obj runtime.Object) error {
	t, ok := obj.(*v1alpha1.Tenant)
	if !ok {
		return apierrors.NewBadRequest("invalid object type")
	}

	return validateTenantName(t.Name)
}

func (s *tenantStrategy) ValidateUpdate(_ context.Context, obj, _ runtime.Object) error {
	t, ok := obj.(*v1alpha1.Tenant)
	if !ok {
		return apierrors.NewBadRequest("invalid object type")
	}

	return validateTenantName(t.Name)
}

func validateTenantName(name string) error {
	if !IsReservedName(name) {
		return nil
	}

	return apierrors.NewBadRequest(fmt.Sprintf(
		"metadata.name %q: %s; this name is reserved and cannot be used as a Tenant",
		name, ReservedReason(name),
	))
}
