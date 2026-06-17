package tenantnamespace

const (
	LabelManagedBy  = "sgroups.io/managed-by"
	ManagedByValue  = "tenant-controller"
	LabelTenantUID  = "sgroups.io/tenant-uid"
	LabelTenantName = "sgroups.io/tenant-name"

	AnnotationCreatedByController = "sgroups.io/created-by-controller"
	AnnotationAdoptedAt           = "sgroups.io/adopted-at"
	AnnotationNamespaceReady      = "sgroups.io/namespace-ready"
	AnnotationNamespaceMessage    = "sgroups.io/namespace-message"

	ReasonNamespaceCreated     = "NamespaceCreated"
	ReasonNamespaceAdopted     = "NamespaceAdopted"
	ReasonNamespaceForbidden   = "NamespaceForbidden"
	ReasonOrphanedNsAdopted    = "OrphanedNamespaceAdopted"
	ReasonOwnerRefRestored     = "OwnerRefRestored"
	ReasonReconcileFailed      = "ReconcileFailed"
	ReasonAwaitingTermination  = "AwaitingNamespaceTermination"
	ReasonNamespaceReadyValue  = "True"
	ReasonNamespaceFailedValue = "False"
)
