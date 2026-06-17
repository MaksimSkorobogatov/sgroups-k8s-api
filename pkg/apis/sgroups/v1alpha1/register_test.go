package v1alpha1_test

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// TestInternalVersionRegistered guards against the regression where strategic-merge
// patch (`kubectl apply` on existing objects) failed with
// `no kind "X" is registered for the internal version of group "sgroups.io"`.
//
// k8s.io/apiserver/pkg/endpoints/installer.go hard-codes
// HubGroupVersion = {group}/__internal and patch.go converts the patched object
// to that version. If the type isn't registered there, the conversion errors out.
func TestInternalVersionRegistered(t *testing.T) {
	t.Parallel()

	internalGV := schema.GroupVersion{
		Group:   v1alpha1.GroupName,
		Version: runtime.APIVersionInternal,
	}

	for _, obj := range v1alpha1.KnownTypes() {
		gvks, _, err := v1alpha1.Scheme.ObjectKinds(obj)
		if err != nil {
			t.Fatalf("ObjectKinds(%T) returned error: %v", obj, err)
		}

		var hasVersioned, hasInternal bool
		for _, gvk := range gvks {
			switch gvk.GroupVersion() {
			case v1alpha1.SchemeGroupVersion:
				hasVersioned = true
			case internalGV:
				hasInternal = true
			}
		}
		if !hasVersioned {
			t.Errorf("%T: missing v1alpha1 registration; got GVKs=%v", obj, gvks)
		}
		if !hasInternal {
			t.Errorf("%T: missing internal registration; got GVKs=%v", obj, gvks)
		}
	}
}

// TestConvertToInternalVersion exercises the exact code path that strategic-merge
// patch uses: scheme.ConvertToVersion(obj, hubGroupVersion). With both versioned
// and internal registered for the same Go type, ConvertToVersion succeeds via the
// fast-path in apimachinery scheme.go (matching kinds, no converter invocation).
//
// Note: setTargetKind intentionally clears the TypeMeta when the target is the
// internal version — internal must not appear on the wire — so we don't assert on
// the returned GVK, only on success and type identity.
func TestConvertToInternalVersion(t *testing.T) {
	t.Parallel()

	internalGV := schema.GroupVersion{
		Group:   v1alpha1.GroupName,
		Version: runtime.APIVersionInternal,
	}

	for _, obj := range v1alpha1.KnownTypes() {
		out, err := v1alpha1.Scheme.ConvertToVersion(obj, internalGV)
		if err != nil {
			t.Errorf("ConvertToVersion(%T, %s) failed: %v", obj, internalGV, err)
			continue
		}
		// Conversion must preserve the Go type (since internal == versioned struct).
		// Anything else would mean apimachinery silently invoked a non-identity converter.
		if gotType, wantType := reflect.TypeOf(out), reflect.TypeOf(obj); gotType != wantType {
			t.Errorf("ConvertToVersion(%T): expected same Go type back, got %T", obj, out)
		}
	}
}
