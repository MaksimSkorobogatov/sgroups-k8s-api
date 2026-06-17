package tenant

import (
	"context"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func newTenant(name string) *v1alpha1.Tenant {
	return &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func TestTenantStrategy_AcceptsRegularNames(t *testing.T) {
	t.Parallel()

	for _, n := range []string{"acme", "tenant-1", "team-platform-prod", "abc"} {
		if err := (&tenantStrategy{}).ValidateCreate(context.Background(), newTenant(n)); err != nil {
			t.Errorf("ValidateCreate(%q) = %v, want nil", n, err)
		}
		if err := (&tenantStrategy{}).ValidateUpdate(context.Background(), newTenant(n), newTenant(n)); err != nil {
			t.Errorf("ValidateUpdate(%q) = %v, want nil", n, err)
		}
	}
}

func TestTenantStrategy_RejectsReservedNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		wantContain string // substring expected in the error message
	}{
		{"default", "reserved"},
		{"kube-system", "kube-"},
		{"kube-public", "kube-"},
		{"kube-node-lease", "kube-"},
		{"kube-foo", "kube-"},
		{"sgroups-system", "sgroups-"},
		{"sgroups-anything", "sgroups-"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (&tenantStrategy{}).ValidateCreate(context.Background(), newTenant(tc.name))
			if err == nil {
				t.Fatalf("ValidateCreate(%q) returned nil, want error", tc.name)
			}
			if !apierrors.IsBadRequest(err) {
				t.Errorf("ValidateCreate(%q): want BadRequest, got %T: %v", tc.name, err, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, "metadata.name") {
				t.Errorf("error message must reference metadata.name; got: %s", msg)
			}
			if !strings.Contains(msg, tc.name) {
				t.Errorf("error message must include the rejected name %q; got: %s", tc.name, msg)
			}
			if !strings.Contains(msg, tc.wantContain) {
				t.Errorf("error message must explain why %q is reserved (contain %q); got: %s",
					tc.name, tc.wantContain, msg)
			}
		})
	}
}

func TestTenantStrategy_RejectsWrongType(t *testing.T) {
	t.Parallel()

	err := (&tenantStrategy{}).ValidateCreate(context.Background(), &v1alpha1.AddressGroup{})
	if err == nil {
		t.Fatal("want error for wrong object type, got nil")
	}
	if !apierrors.IsBadRequest(err) {
		t.Errorf("want BadRequest, got %T: %v", err, err)
	}
}
