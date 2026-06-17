package convert

import (
	common "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// HostToProto converts Host to protobuf.
func HostToProto(in *v1alpha1.Host) *sgroupsv1.Host {
	if in == nil {
		return nil
	}

	return &sgroupsv1.Host{
		Metadata: objectMetaToProto(in.ObjectMeta),
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: in.Spec.DisplayName,
			Comment:     in.Spec.Comment,
			Description: in.Spec.Description,
		},
	}
}

// HostFromProto converts protobuf to Host.
func HostFromProto(in *sgroupsv1.Host) *v1alpha1.Host {
	if in == nil {
		return nil
	}
	out := &v1alpha1.Host{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.KindHost,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Spec: v1alpha1.HostSpec{
			DisplayName: in.GetSpec().GetDisplayName(),
			Comment:     in.GetSpec().GetComment(),
			Description: in.GetSpec().GetDescription(),
		},
		IPs:       hostIPsFromProto(in.GetSpec().GetIps()),
		MetaInfo:  hostMetaInfoFromProto(in.GetSpec().GetMetaInfo()),
		Endpoints: hostEndpointsFromProto(in.GetSpec().GetEndpoints()),
	}
	objectMetaFromProto(&out.ObjectMeta, in.GetMetadata())

	return out
}

// HostFromProtoExt converts list/watch protobuf payload to Host.
func HostFromProtoExt(in *sgroupsv1.HostResp_HostExt) *v1alpha1.Host {
	if in == nil {
		return nil
	}
	out := &v1alpha1.Host{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.KindHost,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Spec: v1alpha1.HostSpec{
			DisplayName: in.GetSpec().GetDisplayName(),
			Comment:     in.GetSpec().GetComment(),
			Description: in.GetSpec().GetDescription(),
		},
		Refs:      ResourceRefsFromProto(in.GetRefs()),
		IPs:       hostIPsFromProto(in.GetSpec().GetIps()),
		MetaInfo:  hostMetaInfoFromProto(in.GetSpec().GetMetaInfo()),
		Endpoints: hostEndpointsFromProto(in.GetSpec().GetEndpoints()),
	}
	objectMetaFromProto(&out.ObjectMeta, in.GetMetadata())

	return out
}

func hostIPsFromProto(in *common.IPs) v1alpha1.HostIPs {
	if in == nil {
		return v1alpha1.HostIPs{}
	}

	return v1alpha1.HostIPs{
		IPv4: in.GetIpv4(),
		IPv6: in.GetIpv6(),
	}
}

func hostMetaInfoFromProto(in *sgroupsv1.Host_Spec_MetaInfo) v1alpha1.HostMetaInfo {
	if in == nil {
		return v1alpha1.HostMetaInfo{}
	}

	return v1alpha1.HostMetaInfo{
		HostName:        in.GetHostName(),
		OS:              in.GetOs(),
		Platform:        in.GetPlatform(),
		PlatformFamily:  in.GetPlatformFamily(),
		PlatformVersion: in.GetPlatformVersion(),
		KernelVersion:   in.GetKernelVersion(),
	}
}

func hostEndpointsFromProto(in *sgroupsv1.Host_Spec_Endpoints) v1alpha1.HostEndpoints {
	if in == nil {
		return v1alpha1.HostEndpoints{}
	}

	ports := in.GetPorts()
	out := v1alpha1.HostEndpoints{Address: in.GetAddress()}
	if len(ports) > 0 {
		out.Ports = make([]v1alpha1.HostEndpointPort, 0, len(ports))
		for _, p := range ports {
			if p == nil {
				continue
			}
			out.Ports = append(out.Ports, v1alpha1.HostEndpointPort{
				Name: p.GetName(),
				Port: p.GetPort(),
			})
		}
	}

	return out
}
