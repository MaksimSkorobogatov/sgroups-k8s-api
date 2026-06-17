package tenantnamespace

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

const forbiddenTenantRequeueAfter = 5 * time.Minute

// Reconciler synchronises a Namespace with its same-named Tenant.
type Reconciler struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Now      func() time.Time // test seam; defaults to time.Now
}

func (r *Reconciler) SetupWithManager(mgr manager.Manager) error {
	if r.Now == nil {
		r.Now = time.Now
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("tenant-namespace").
		For(&v1alpha1.Tenant{}, builder.WithPredicates()).
		Owns(&corev1.Namespace{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("tenant", req.Name)

	tenantObj := &v1alpha1.Tenant{}
	if err := r.Client.Get(ctx, req.NamespacedName, tenantObj); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("get tenant: %w", err)
	}

	if !tenantObj.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	if isReservedTenantName(tenantObj.Name) {
		msg := fmt.Sprintf("tenant name %q is reserved; reconciler refuses to manage a namespace for it", tenantObj.Name)
		logger.Info("refusing to reconcile tenant with reserved name")
		if err := r.patchTenantStatus(ctx, tenantObj, ReasonNamespaceFailedValue, msg); err != nil {
			return reconcile.Result{}, fmt.Errorf("patch tenant status: %w", err)
		}
		r.recordEvent(tenantObj, corev1.EventTypeWarning, ReasonNamespaceForbidden, msg)

		return reconcile.Result{RequeueAfter: forbiddenTenantRequeueAfter}, nil
	}

	outcome, err := r.ensureNamespace(ctx, tenantObj)
	if err != nil {
		statusErr := r.patchTenantStatus(ctx, tenantObj, ReasonNamespaceFailedValue, err.Error())
		r.recordEvent(tenantObj, corev1.EventTypeWarning, ReasonReconcileFailed, err.Error())
		if statusErr != nil {
			return reconcile.Result{}, fmt.Errorf("patch tenant status after %s: %w", err.Error(), statusErr)
		}

		return reconcile.Result{}, err
	}

	if err := r.patchTenantStatus(ctx, tenantObj, ReasonNamespaceReadyValue, outcome.message); err != nil {
		return reconcile.Result{}, fmt.Errorf("patch tenant status: %w", err)
	}
	if outcome.eventReason != "" {
		r.recordEvent(tenantObj, outcome.eventType, outcome.eventReason, outcome.message)
	}
	if outcome.requeueAfter > 0 {
		return reconcile.Result{RequeueAfter: outcome.requeueAfter}, nil
	}

	return reconcile.Result{}, nil
}

type ensureOutcome struct {
	message      string
	eventReason  string
	eventType    string
	requeueAfter time.Duration
}

func (r *Reconciler) ensureNamespace(ctx context.Context, t *v1alpha1.Tenant) (ensureOutcome, error) {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: t.Name}, ns)
	switch {
	case apierrors.IsNotFound(err):
		fresh := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: t.Name,
				Annotations: map[string]string{
					AnnotationCreatedByController: "true",
				},
			},
		}
		applyManagedLabels(fresh, t)
		applyOwnerReference(fresh, t)
		if err := r.Client.Create(ctx, fresh); err != nil {
			return ensureOutcome{}, fmt.Errorf("create namespace: %w", err)
		}
		logger.Info("namespace created", "namespace", t.Name)

		return ensureOutcome{
			message:     "namespace created",
			eventReason: ReasonNamespaceCreated,
			eventType:   corev1.EventTypeNormal,
		}, nil

	case err != nil:
		return ensureOutcome{}, fmt.Errorf("get namespace: %w", err)
	}

	if !ns.DeletionTimestamp.IsZero() {
		logger.Info("namespace is terminating; will retry", "namespace", t.Name)

		return ensureOutcome{
			message:      "namespace is terminating; awaiting deletion before recreate",
			eventReason:  ReasonAwaitingTermination,
			eventType:    corev1.EventTypeNormal,
			requeueAfter: 10 * time.Second,
		}, nil
	}

	patch := client.MergeFrom(ns.DeepCopy())

	switch {
	case matchesTenant(ns, t):
		if !hasOwnerReferenceTo(ns, t) {
			applyOwnerReference(ns, t)
			logger.Info("restoring owner reference on managed namespace", "namespace", ns.Name)
			if err := r.Client.Patch(ctx, ns, patch); err != nil {
				return ensureOutcome{}, fmt.Errorf("patch namespace owner ref: %w", err)
			}

			return ensureOutcome{
				message:     "owner reference restored",
				eventReason: ReasonOwnerRefRestored,
				eventType:   corev1.EventTypeNormal,
			}, nil
		}

		return ensureOutcome{message: "namespace ready"}, nil

	case isManagedByController(ns):
		applyManagedLabels(ns, t)
		applyOwnerReference(ns, t)
		setAnnotation(ns, AnnotationAdoptedAt, r.Now().UTC().Format(time.RFC3339))
		logger.Info("re-linking orphaned namespace to current tenant incarnation",
			"namespace", ns.Name, "newUID", t.UID)
		if err := r.Client.Patch(ctx, ns, patch); err != nil {
			return ensureOutcome{}, fmt.Errorf("patch orphaned namespace: %w", err)
		}

		return ensureOutcome{
			message:     "orphaned namespace re-adopted",
			eventReason: ReasonOrphanedNsAdopted,
			eventType:   corev1.EventTypeNormal,
		}, nil

	default:
		applyManagedLabels(ns, t)
		applyOwnerReference(ns, t)
		setAnnotation(ns, AnnotationAdoptedAt, r.Now().UTC().Format(time.RFC3339))
		logger.Info("adopting pre-existing namespace", "namespace", ns.Name)
		if err := r.Client.Patch(ctx, ns, patch); err != nil {
			return ensureOutcome{}, fmt.Errorf("patch namespace for adopt: %w", err)
		}

		return ensureOutcome{
			message:     "namespace adopted",
			eventReason: ReasonNamespaceAdopted,
			eventType:   corev1.EventTypeNormal,
		}, nil
	}
}

func (r *Reconciler) patchTenantStatus(ctx context.Context, t *v1alpha1.Tenant, ready, message string) error {
	current := t.Annotations
	wantMessage := truncateMessage(message)
	if current[AnnotationNamespaceReady] == ready && current[AnnotationNamespaceMessage] == wantMessage {
		return nil
	}

	patch := client.MergeFrom(t.DeepCopy())
	if t.Annotations == nil {
		t.Annotations = map[string]string{}
	}
	t.Annotations[AnnotationNamespaceReady] = ready
	t.Annotations[AnnotationNamespaceMessage] = wantMessage

	return r.Client.Patch(ctx, t, patch)
}

func (r *Reconciler) recordEvent(t *v1alpha1.Tenant, eventType, reason, message string) {
	if r.Recorder == nil {
		return
	}
	r.Recorder.Event(t, eventType, reason, message)
}

func setAnnotation(obj metav1.Object, key, value string) {
	a := obj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
	}
	a[key] = value
	obj.SetAnnotations(a)
}

const maxMessageLen = 1024

func truncateMessage(s string) string {
	if len(s) <= maxMessageLen {
		return s
	}

	return s[:maxMessageLen-3] + "..."
}
