package tenantnamespace

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func fixedClock() time.Time {
	return time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
}

const fixedTimestamp = "2026-05-08T12:00:00Z"

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("corev1.AddToScheme: %v", err)
	}
	if err := v1alpha1.AddToWireScheme(s); err != nil {
		t.Fatalf("v1alpha1.AddToWireScheme: %v", err)
	}

	return s
}

func newReconciler(t *testing.T, objs ...client.Object) (*Reconciler, *record.FakeRecorder) {
	t.Helper()
	s := newScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		Build()
	rec := record.NewFakeRecorder(16)

	return &Reconciler{
		Client:   c,
		Scheme:   s,
		Recorder: rec,
		Now:      fixedClock,
	}, rec
}

func mkTenant(name string, uid types.UID) *v1alpha1.Tenant {
	return &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: uid},
	}
}

func reqFor(name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}
}

func readNamespace(t *testing.T, r *Reconciler, name string) *corev1.Namespace {
	t.Helper()
	ns := &corev1.Namespace{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: name}, ns); err != nil {
		t.Fatalf("get namespace %q: %v", name, err)
	}

	return ns
}

func readTenant(t *testing.T, r *Reconciler, name string) *v1alpha1.Tenant {
	t.Helper()
	out := &v1alpha1.Tenant{}
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: name}, out); err != nil {
		t.Fatalf("get tenant %q: %v", name, err)
	}

	return out
}

func TestReconcile_TenantNotFound(t *testing.T) {
	t.Parallel()
	r, _ := newReconciler(t)
	res, err := r.Reconcile(context.Background(), reqFor("ghost"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected no requeue for missing tenant, got %v", res.RequeueAfter)
	}
}

func TestReconcile_TenantBeingDeleted(t *testing.T) {
	t.Parallel()
	now := metav1.Now()
	tn := mkTenant("acme", "uid-1")
	tn.DeletionTimestamp = &now
	tn.Finalizers = []string{"placeholder/finalizer"} // fake client requires a finalizer to keep an object with DeletionTimestamp

	r, _ := newReconciler(t, tn)
	res, err := r.Reconcile(context.Background(), reqFor("acme"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got %v", res.RequeueAfter)
	}
	// We did NOT touch annotations on the tenant.
	updated := readTenant(t, r, "acme")
	if _, ok := updated.Annotations[AnnotationNamespaceReady]; ok {
		t.Errorf("expected no namespace-ready annotation set on terminating tenant, got: %v", updated.Annotations)
	}
}

func TestReconcile_ReservedName(t *testing.T) {
	t.Parallel()

	// Pre-existing namespace called "kube-system" must remain UNTOUCHED.
	preExistingKubeSystem := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
	}

	cases := []struct {
		name string
	}{
		{"kube-system"}, {"kube-foo"}, {"sgroups-system"}, {"default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, rec := newReconciler(t,
				mkTenant(tc.name, "uid-x"),
				preExistingKubeSystem.DeepCopy(),
			)
			res, err := r.Reconcile(context.Background(), reqFor(tc.name))
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if res.RequeueAfter == 0 {
				t.Errorf("expected slow requeue for reserved tenant, got 0")
			}

			// Warning event must mention the forbidden reason.
			select {
			case ev := <-rec.Events:
				if want := ReasonNamespaceForbidden; !contains(ev, want) {
					t.Errorf("expected event to contain %q, got %q", want, ev)
				}
			default:
				t.Errorf("expected NamespaceForbidden event")
			}

			// Tenant annotated as ready=False with reason in message.
			updated := readTenant(t, r, tc.name)
			if got := updated.Annotations[AnnotationNamespaceReady]; got != ReasonNamespaceFailedValue {
				t.Errorf("expected ready=False, got %q", got)
			}

			if tc.name == "kube-system" {
				ns := readNamespace(t, r, "kube-system")
				if isManagedByController(ns) {
					t.Errorf("kube-system was incorrectly adopted; labels: %v", ns.Labels)
				}
				if len(ns.OwnerReferences) != 0 {
					t.Errorf("kube-system should have no owner refs, got: %v", ns.OwnerReferences)
				}
			}
		})
	}
}

func TestReconcile_CreateNamespace(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-1")
	r, rec := newReconciler(t, tn)

	res, err := r.Reconcile(context.Background(), reqFor("acme"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected no requeue on success, got %v", res.RequeueAfter)
	}

	ns := readNamespace(t, r, "acme")
	// Labels.
	if got := ns.Labels[LabelManagedBy]; got != ManagedByValue {
		t.Errorf("managed-by = %q, want %q", got, ManagedByValue)
	}
	if got := ns.Labels[LabelTenantUID]; got != "uid-1" {
		t.Errorf("tenant-uid = %q, want %q", got, "uid-1")
	}
	if got := ns.Labels[LabelTenantName]; got != "acme" {
		t.Errorf("tenant-name = %q, want %q", got, "acme")
	}
	// Annotation: created vs adopted
	if got := ns.Annotations[AnnotationCreatedByController]; got != "true" {
		t.Errorf("created-by-controller annotation missing/wrong: %q", got)
	}
	if _, ok := ns.Annotations[AnnotationAdoptedAt]; ok {
		t.Errorf("created namespace must not have adopted-at annotation; got %q", ns.Annotations[AnnotationAdoptedAt])
	}
	// Owner reference.
	if !hasOwnerReferenceTo(ns, tn) {
		t.Errorf("expected owner ref to %s/%s; got %v", tn.Name, tn.UID, ns.OwnerReferences)
	}
	for i := range ns.OwnerReferences {
		ref := &ns.OwnerReferences[i]
		if ref.UID == tn.UID {
			if ref.Controller == nil || !*ref.Controller {
				t.Errorf("owner ref must have controller=true")
			}
			if ref.BlockOwnerDeletion == nil || *ref.BlockOwnerDeletion {
				t.Errorf("owner ref must have blockOwnerDeletion=false")
			}
		}
	}

	// Tenant annotations reflect Ready.
	updated := readTenant(t, r, "acme")
	if got := updated.Annotations[AnnotationNamespaceReady]; got != ReasonNamespaceReadyValue {
		t.Errorf("tenant ready = %q, want %q", got, ReasonNamespaceReadyValue)
	}

	// Event emitted.
	select {
	case ev := <-rec.Events:
		if !contains(ev, ReasonNamespaceCreated) {
			t.Errorf("expected NamespaceCreated event, got %q", ev)
		}
	default:
		t.Errorf("expected event")
	}
}

func TestReconcile_AdoptNamespace(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-1")
	preExisting := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "acme",
			Labels:      map[string]string{"team": "platform"}, // user labels must survive
			Annotations: map[string]string{"contact": "platform@acme.io"},
		},
	}
	r, rec := newReconciler(t, tn, preExisting)

	if _, err := r.Reconcile(context.Background(), reqFor("acme")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	ns := readNamespace(t, r, "acme")

	// Adopt path: managed-by labels added, user labels survived.
	if !isManagedByController(ns) {
		t.Errorf("namespace not adopted, labels: %v", ns.Labels)
	}
	if got := ns.Labels["team"]; got != "platform" {
		t.Errorf("user label 'team' lost: got %q", got)
	}
	// Adopt path: adopted-at set, created-by-controller NOT set.
	if got := ns.Annotations[AnnotationAdoptedAt]; got != fixedTimestamp {
		t.Errorf("adopted-at = %q, want %q", got, fixedTimestamp)
	}
	if _, ok := ns.Annotations[AnnotationCreatedByController]; ok {
		t.Errorf("adopted namespace must not have created-by-controller annotation")
	}
	if got := ns.Annotations["contact"]; got != "platform@acme.io" {
		t.Errorf("user annotation lost; got %q", got)
	}
	// Owner ref installed.
	if !hasOwnerReferenceTo(ns, tn) {
		t.Errorf("owner ref missing after adopt: %v", ns.OwnerReferences)
	}

	// Event was NamespaceAdopted, not NamespaceCreated.
	select {
	case ev := <-rec.Events:
		if !contains(ev, ReasonNamespaceAdopted) {
			t.Errorf("expected NamespaceAdopted event, got %q", ev)
		}
	default:
		t.Errorf("expected event")
	}
}

func TestReconcile_SteadyStateNoOp(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-1")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "acme",
			ResourceVersion: "100",
			Labels: map[string]string{
				LabelManagedBy:  ManagedByValue,
				LabelTenantUID:  string(tn.UID),
				LabelTenantName: tn.Name,
			},
			Annotations: map[string]string{AnnotationCreatedByController: "true"},
			OwnerReferences: []metav1.OwnerReference{
				desiredOwnerReference(tn),
			},
		},
	}
	r, _ := newReconciler(t, tn, ns)

	if _, err := r.Reconcile(context.Background(), reqFor("acme")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	got := readNamespace(t, r, "acme")
	if got.ResourceVersion != "100" {
		t.Errorf("namespace was unexpectedly mutated; resourceVersion %q != \"100\"", got.ResourceVersion)
	}
}

func TestReconcile_RestoresOwnerReference(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-1")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acme",
			Labels: map[string]string{
				LabelManagedBy:  ManagedByValue,
				LabelTenantUID:  string(tn.UID),
				LabelTenantName: tn.Name,
			},
			// NO owner references.
		},
	}
	r, rec := newReconciler(t, tn, ns)
	if _, err := r.Reconcile(context.Background(), reqFor("acme")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got := readNamespace(t, r, "acme")
	if !hasOwnerReferenceTo(got, tn) {
		t.Errorf("owner reference not restored: %v", got.OwnerReferences)
	}
	select {
	case ev := <-rec.Events:
		if !contains(ev, ReasonOwnerRefRestored) {
			t.Errorf("expected OwnerRefRestored event, got %q", ev)
		}
	default:
		t.Errorf("expected event")
	}
}

func TestReconcile_OrphanedNamespaceUIDMismatch(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-NEW")
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acme",
			Labels: map[string]string{
				LabelManagedBy:  ManagedByValue,
				LabelTenantUID:  "uid-OLD",
				LabelTenantName: "acme",
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.KindTenant,
				Name:       "acme",
				UID:        "uid-OLD",
			}},
		},
	}
	r, rec := newReconciler(t, tn, ns)
	if _, err := r.Reconcile(context.Background(), reqFor("acme")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got := readNamespace(t, r, "acme")
	if got.Labels[LabelTenantUID] != "uid-NEW" {
		t.Errorf("tenant-uid not updated: %q", got.Labels[LabelTenantUID])
	}
	if !hasOwnerReferenceTo(got, tn) {
		t.Errorf("expected new owner ref, got: %v", got.OwnerReferences)
	}
	// There must be exactly one Tenant owner ref (not two).
	count := 0
	for _, ref := range got.OwnerReferences {
		if ref.Kind == v1alpha1.KindTenant {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 Tenant owner ref, got %d (%v)", count, got.OwnerReferences)
	}
	if got.Annotations[AnnotationAdoptedAt] != fixedTimestamp {
		t.Errorf("adopted-at not refreshed: %q", got.Annotations[AnnotationAdoptedAt])
	}
	select {
	case ev := <-rec.Events:
		if !contains(ev, ReasonOrphanedNsAdopted) {
			t.Errorf("expected OrphanedNamespaceAdopted, got %q", ev)
		}
	default:
		t.Errorf("expected event")
	}
}

func TestReconcile_NamespaceTerminating(t *testing.T) {
	t.Parallel()
	tn := mkTenant("acme", "uid-NEW")
	now := metav1.Now()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "acme",
			DeletionTimestamp: &now,
			Finalizers:        []string{"kubernetes"}, // namespaces always have this finalizer in real life
		},
	}
	r, _ := newReconciler(t, tn, ns)
	res, err := r.Reconcile(context.Background(), reqFor("acme"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Errorf("expected requeue while namespace terminates, got 0")
	}
	// Must not have been mutated (no labels added).
	got := readNamespace(t, r, "acme")
	if isManagedByController(got) {
		t.Errorf("must not adopt a terminating namespace")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}
