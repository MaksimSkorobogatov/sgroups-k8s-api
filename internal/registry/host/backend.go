package host

import (
	"context"
	"errors"

	"github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"sgroups.io/sgroups-k8s-api/internal/registry/base"
	"sgroups.io/sgroups-k8s-api/internal/registry/convert"
	regerrors "sgroups.io/sgroups-k8s-api/internal/registry/errors"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

type backend struct {
	client *client.Client
}

func (b *backend) NamespaceScoped() bool {
	return true
}

func (b *backend) Resource() schema.GroupResource {
	return v1alpha1.Resource(v1alpha1.ResourceHosts)
}

func (b *backend) List(ctx context.Context, sel base.Selection) (*v1alpha1.HostList, error) {
	resSelector := &common.ResSelector{
		LabelSelector: sel.Labels,
	}
	if sel.Name != "" || sel.Namespace != "" {
		resSelector.FieldSelector = &common.FieldSelector{
			Name:      sel.Name,
			Namespace: sel.Namespace,
		}
	}

	req := &sgroupsv1.HostReq_List{Selectors: []*common.ResSelector{resSelector}}
	resp, err := b.client.Hosts.List(ctx, req)
	if err != nil {
		return nil, regerrors.FromGRPC(err, b.Resource(), sel.Name)
	}

	out := &v1alpha1.HostList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.KindHostList,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Items: make([]v1alpha1.Host, 0, len(resp.GetHosts())),
	}
	out.ResourceVersion = resp.GetResourceVersion()
	for _, h := range resp.GetHosts() {
		if converted := convert.HostFromProtoExt(h); converted != nil {
			out.Items = append(out.Items, *converted)
		}
	}

	return out, nil
}

func (b *backend) Upsert(ctx context.Context, obj *v1alpha1.Host) (*v1alpha1.Host, error) {
	req := &sgroupsv1.HostReq_Upsert{
		Hosts: []*sgroupsv1.Host{convert.HostToProto(obj)},
	}
	resp, err := b.client.Hosts.Upsert(ctx, req)
	if err != nil {
		return nil, regerrors.FromGRPC(err, b.Resource(), obj.Name)
	}
	if len(resp.GetHosts()) == 0 {
		return nil, apierrors.NewInternalError(errors.New("empty upsert response"))
	}

	return convert.HostFromProto(resp.GetHosts()[0]), nil
}

func (b *backend) Delete(ctx context.Context, name, namespace string) error {
	req := &sgroupsv1.HostReq_Delete{
		Hosts: []*sgroupsv1.HostReq_Delete_Host{
			{
				Metadata: &common.MetadataScope{
					Name:      name,
					Namespace: namespace,
				},
			},
		},
	}
	_, err := b.client.Hosts.Delete(ctx, req)
	if err != nil {
		return regerrors.FromGRPC(err, b.Resource(), name)
	}

	return nil
}

func (b *backend) Watch(ctx context.Context, sel base.Selection, resourceVersion string) (watch.Interface, error) {
	resSelector := &common.ResSelector{
		LabelSelector: sel.Labels,
	}
	if sel.Name != "" || sel.Namespace != "" {
		resSelector.FieldSelector = &common.FieldSelector{
			Name:      sel.Name,
			Namespace: sel.Namespace,
		}
	}

	req := &sgroupsv1.HostReq_Watch{
		ResourceVersion: resourceVersion,
		Selectors:       []*common.ResSelector{resSelector},
	}
	ctx, cancel := context.WithCancel(ctx)
	stream, err := b.client.Hosts.Watch(ctx, req)
	if err != nil {
		cancel()

		return nil, regerrors.FromGRPC(err, b.Resource(), sel.Name)
	}

	return base.NewStreamWatch(
		cancel,
		stream.Recv,
		func(evt *sgroupsv1.HostResp_Watch) common.WatchEventType {
			return evt.GetType()
		},
		func(evt *sgroupsv1.HostResp_Watch) []*sgroupsv1.HostResp_HostExt {
			return evt.GetHosts()
		},
		func(h *sgroupsv1.HostResp_HostExt) runtime.Object {
			return convert.HostFromProtoExt(h)
		},
	), nil
}
