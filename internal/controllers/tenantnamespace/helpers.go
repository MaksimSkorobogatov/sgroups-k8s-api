package tenantnamespace

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sgroups.io/sgroups-k8s-api/internal/registry/tenant"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func isReservedTenantName(name string) bool {
	return tenant.IsReservedName(name)
}

func isManagedByController(ns *corev1.Namespace) bool {
	if ns == nil || ns.Labels == nil {
		return false
	}
	_, ok := ns.Labels[LabelManagedBy]

	return ok
}

func matchesTenant(ns *corev1.Namespace, t *v1alpha1.Tenant) bool {
	if !isManagedByController(ns) {
		return false
	}

	return ns.Labels[LabelTenantUID] == string(t.UID)
}

func hasOwnerReferenceTo(ns *corev1.Namespace, t *v1alpha1.Tenant) bool {
	if ns == nil || t == nil {
		return false
	}
	for i := range ns.OwnerReferences {
		ref := &ns.OwnerReferences[i]
		if ref.APIVersion == v1alpha1.SchemeGroupVersion.String() &&
			ref.Kind == v1alpha1.KindTenant &&
			ref.Name == t.Name &&
			ref.UID == t.UID {
			return true
		}
	}

	return false
}

func desiredOwnerReference(t *v1alpha1.Tenant) metav1.OwnerReference {
	controller := true
	blockOwnerDeletion := false

	return metav1.OwnerReference{
		APIVersion:         v1alpha1.SchemeGroupVersion.String(),
		Kind:               v1alpha1.KindTenant,
		Name:               t.Name,
		UID:                t.UID,
		Controller:         &controller,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// applyOwnerReference upserts the controller owner ref; same-name refs are replaced.
func applyOwnerReference(ns *corev1.Namespace, t *v1alpha1.Tenant) {
	desired := desiredOwnerReference(t)
	for i := range ns.OwnerReferences {
		ref := &ns.OwnerReferences[i]
		if ref.APIVersion == v1alpha1.SchemeGroupVersion.String() &&
			ref.Kind == v1alpha1.KindTenant &&
			ref.Name == t.Name {
			ns.OwnerReferences[i] = desired

			return
		}
	}
	ns.OwnerReferences = append(ns.OwnerReferences, desired)
}

func applyManagedLabels(ns *corev1.Namespace, t *v1alpha1.Tenant) {
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[LabelManagedBy] = ManagedByValue
	ns.Labels[LabelTenantUID] = string(t.UID)
	ns.Labels[LabelTenantName] = t.Name
}
