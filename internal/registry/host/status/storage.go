// Package status implements the hosts/status subresource.
// It accepts PATCH/PUT on /apis/sgroups.io/v1alpha1/namespaces/{ns}/hosts/{name}/status
// and proxies the healthy field to sg-server via UpdHealthStatus gRPC.
package status

import (
	"context"
	"errors"
	"fmt"

	common "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"sgroups.io/sgroups-k8s-api/internal/registry/convert"
	regerrors "sgroups.io/sgroups-k8s-api/internal/registry/errors"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

// Storage implements the hosts/status subresource.
type Storage struct {
	client *client.Client
}

// NewStorage creates a new status subresource storage.
func NewStorage(c *client.Client, _ registryoptions.StorageOptions) *Storage {
	return &Storage{client: c}
}

var (
	_ rest.Storage = (*Storage)(nil)
	_ rest.Scoper  = (*Storage)(nil)
	_ rest.Updater = (*Storage)(nil)
	_ rest.Getter  = (*Storage)(nil)
	_ rest.Patcher = (*Storage)(nil)
)

func (s *Storage) NamespaceScoped() bool { return true }
func (s *Storage) New() runtime.Object   { return &v1alpha1.Host{} }
func (s *Storage) Destroy()              {}

// Get returns the current Host (required by rest.Patcher).
func (s *Storage) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required")
	}

	req := &sgroupsv1.HostReq_List{
		Selectors: []*common.ResSelector{
			{
				FieldSelector: &common.FieldSelector{
					Name:      name,
					Namespace: ns,
				},
			},
		},
	}
	resp, err := s.client.Hosts.List(ctx, req)
	if err != nil {
		return nil, regerrors.FromGRPC(err, v1alpha1.Resource(v1alpha1.ResourceHosts), name)
	}
	if len(resp.GetHosts()) == 0 {
		return nil, apierrors.NewNotFound(v1alpha1.Resource(v1alpha1.ResourceHosts), name)
	}

	return convert.HostFromProtoExt(resp.GetHosts()[0]), nil
}

// Update applies the status update (healthy field) via UpdHealthStatus gRPC.
func (s *Storage) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions,
) (runtime.Object, bool, error) {
	ns, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, false, apierrors.NewBadRequest("namespace is required")
	}

	oldObj, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}

	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	newHost, ok := newObj.(*v1alpha1.Host)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected *Host, got %T", newObj))
	}

	oldHost, ok := oldObj.(*v1alpha1.Host)
	if !ok {
		return nil, false, apierrors.NewBadRequest(fmt.Sprintf("expected *Host, got %T", oldObj))
	}

	if err := updateValidation(ctx, newObj, oldObj); err != nil {
		return nil, false, err
	}

	updReq := &sgroupsv1.HostReq_UpdHealthStatus{
		Hosts: []*sgroupsv1.HostReq_UpdHealthStatus_Host{
			{
				Metadata: &common.MetadataScope{
					Uid:       string(oldHost.UID),
					Name:      name,
					Namespace: ns,
				},
				Spec: &sgroupsv1.HostReq_UpdHealthStatus_Host_Spec{
					Healthy: convert.HealthyBoolToProto(newHost.Healthy),
				},
			},
		},
	}
	resp, err := s.client.Hosts.UpdHealthStatus(ctx, updReq)
	if err != nil {
		return nil, false, regerrors.FromGRPC(err, v1alpha1.Resource(v1alpha1.ResourceHosts), name)
	}
	if len(resp.GetHosts()) == 0 {
		return nil, false, apierrors.NewInternalError(errors.New("empty upd-health-status response"))
	}

	return convert.HostFromProto(resp.GetHosts()[0]), false, nil
}
