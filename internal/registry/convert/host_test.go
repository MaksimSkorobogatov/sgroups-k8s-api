package convert

import (
	"testing"
	"time"

	commonpb "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestHostConversion(t *testing.T) {
	tests := []struct {
		name string
		in   *v1alpha1.Host
	}{
		{
			name: "minimal",
			in: &v1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{Name: "host-1", Namespace: "default"},
			},
		},
		{
			name: "full spec",
			in: &v1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{Name: "host-full", Namespace: "prod"},
				Spec: v1alpha1.HostSpec{
					DisplayName: "Host Full",
					Comment:     "comment",
					Description: "description",
				},
			},
		},
		{
			name: "with labels annotations uid rv ts",
			in: &v1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "host-meta",
					Namespace:         "staging",
					UID:               types.UID("uid-host-1"),
					ResourceVersion:   "23",
					CreationTimestamp: metav1.NewTime(time.Date(2025, 4, 11, 8, 30, 0, 0, time.UTC)),
					Labels:            map[string]string{"env": "staging"},
					Annotations:       map[string]string{"note": "x"},
				},
				Spec: v1alpha1.HostSpec{
					DisplayName: "Meta host",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := HostToProto(tt.in)
			require.NotNil(t, pb)

			got := HostFromProto(pb)
			require.NotNil(t, got)

			require.Equal(t, tt.in.Name, got.Name)
			require.Equal(t, tt.in.Namespace, got.Namespace)
			require.Equal(t, tt.in.UID, got.UID)
			require.Equal(t, tt.in.ResourceVersion, got.ResourceVersion)
			require.Equal(t, tt.in.Labels, got.Labels)
			require.Equal(t, tt.in.Annotations, got.Annotations)
			if !tt.in.CreationTimestamp.IsZero() {
				require.True(t, tt.in.CreationTimestamp.Time.Equal(got.CreationTimestamp.Time))
			}

			require.Equal(t, tt.in.Spec.DisplayName, got.Spec.DisplayName)
			require.Equal(t, tt.in.Spec.Comment, got.Spec.Comment)
			require.Equal(t, tt.in.Spec.Description, got.Spec.Description)

			require.Equal(t, v1alpha1.KindHost, got.Kind)
			require.Equal(t, v1alpha1.SchemeGroupVersion.String(), got.APIVersion)
		})
	}
}

func TestHostFromProtoExt(t *testing.T) {
	ts := time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC)

	ext := &sgroupsv1.HostResp_HostExt{
		Metadata: &commonpb.Metadata{
			Uid:               "ext-uid",
			Name:              "host-ext",
			Namespace:         "default",
			ResourceVersion:   "19",
			CreationTimestamp: timestamppb.New(ts),
			Labels:            map[string]string{"env": "test"},
		},
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: "Ext host",
			Comment:     "ext comment",
			Description: "ext description",
		},
	}

	got := HostFromProtoExt(ext)
	require.NotNil(t, got)

	require.Equal(t, "host-ext", got.Name)
	require.Equal(t, "default", got.Namespace)
	require.Equal(t, types.UID("ext-uid"), got.UID)
	require.Equal(t, "19", got.ResourceVersion)
	require.Equal(t, map[string]string{"env": "test"}, got.Labels)
	require.Equal(t, "Ext host", got.Spec.DisplayName)
	require.Equal(t, "ext comment", got.Spec.Comment)
	require.Equal(t, "ext description", got.Spec.Description)
	require.True(t, ts.Equal(got.CreationTimestamp.Time))
}

func TestHostFromProtoWithIPs(t *testing.T) {
	host := &sgroupsv1.Host{
		Metadata: &commonpb.Metadata{Name: "h1", Namespace: "default"},
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: "H1",
			Ips: &commonpb.IPs{
				Ipv4: []string{"10.0.0.1", "10.0.0.2"},
				Ipv6: []string{"fe80::1"},
			},
			MetaInfo: &sgroupsv1.Host_Spec_MetaInfo{
				HostName:        "h1.local",
				Os:              "linux",
				Platform:        "ubuntu",
				PlatformFamily:  "debian",
				PlatformVersion: "22.04",
				KernelVersion:   "5.15.0",
			},
		},
	}

	got := HostFromProto(host)
	require.NotNil(t, got)
	require.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, got.IPs.IPv4)
	require.Equal(t, []string{"fe80::1"}, got.IPs.IPv6)
	require.Equal(t, "h1.local", got.MetaInfo.HostName)
	require.Equal(t, "linux", got.MetaInfo.OS)
	require.Equal(t, "ubuntu", got.MetaInfo.Platform)
	require.Equal(t, "debian", got.MetaInfo.PlatformFamily)
	require.Equal(t, "22.04", got.MetaInfo.PlatformVersion)
	require.Equal(t, "5.15.0", got.MetaInfo.KernelVersion)
}

func TestHostFromProtoWithEndpoints(t *testing.T) {
	host := &sgroupsv1.Host{
		Metadata: &commonpb.Metadata{Name: "h-ep", Namespace: "default"},
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: "H-EP",
			Endpoints: &sgroupsv1.Host_Spec_Endpoints{
				Address: "10.0.0.5",
				Ports: []*sgroupsv1.Host_Spec_Endpoints_Port{
					{Name: "api", Port: 8443},
					{Name: "metrics", Port: 9090},
				},
			},
		},
	}

	got := HostFromProto(host)
	require.NotNil(t, got)
	require.Equal(t, "10.0.0.5", got.Endpoints.Address)
	require.Len(t, got.Endpoints.Ports, 2)
	require.Equal(t, "api", got.Endpoints.Ports[0].Name)
	require.Equal(t, uint32(8443), got.Endpoints.Ports[0].Port)
	require.Equal(t, "metrics", got.Endpoints.Ports[1].Name)
	require.Equal(t, uint32(9090), got.Endpoints.Ports[1].Port)
}

func TestHostFromProtoExtWithEndpoints(t *testing.T) {
	ext := &sgroupsv1.HostResp_HostExt{
		Metadata: &commonpb.Metadata{Name: "h-ep-ext", Namespace: "default"},
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: "H-EP-Ext",
			Endpoints: &sgroupsv1.Host_Spec_Endpoints{
				Address: "", // falls back to peer addr server-side
				Ports: []*sgroupsv1.Host_Spec_Endpoints_Port{
					{Name: "health", Port: 8081},
				},
			},
		},
	}

	got := HostFromProtoExt(ext)
	require.NotNil(t, got)
	require.Empty(t, got.Endpoints.Address)
	require.Len(t, got.Endpoints.Ports, 1)
	require.Equal(t, "health", got.Endpoints.Ports[0].Name)
	require.Equal(t, uint32(8081), got.Endpoints.Ports[0].Port)
}

func TestHostFromProtoNoEndpoints(t *testing.T) {
	host := &sgroupsv1.Host{
		Metadata: &commonpb.Metadata{Name: "h-no-ep", Namespace: "default"},
		Spec:     &sgroupsv1.Host_Spec{DisplayName: "H-NoEP"},
	}

	got := HostFromProto(host)
	require.NotNil(t, got)
	require.Empty(t, got.Endpoints.Address)
	require.Empty(t, got.Endpoints.Ports)
}

func TestHostFromProtoEndpointsNilPortEntry(t *testing.T) {
	host := &sgroupsv1.Host{
		Metadata: &commonpb.Metadata{Name: "h-nil-port", Namespace: "default"},
		Spec: &sgroupsv1.Host_Spec{
			Endpoints: &sgroupsv1.Host_Spec_Endpoints{
				Address: "10.0.0.6",
				Ports: []*sgroupsv1.Host_Spec_Endpoints_Port{
					{Name: "api", Port: 8443},
					nil, // must be skipped
					{Name: "metrics", Port: 9090},
				},
			},
		},
	}

	got := HostFromProto(host)
	require.NotNil(t, got)
	require.Len(t, got.Endpoints.Ports, 2)
	require.Equal(t, "api", got.Endpoints.Ports[0].Name)
	require.Equal(t, "metrics", got.Endpoints.Ports[1].Name)
}

func TestHostFromProtoExtWithIPs(t *testing.T) {
	ext := &sgroupsv1.HostResp_HostExt{
		Metadata: &commonpb.Metadata{Name: "h2", Namespace: "default"},
		Spec: &sgroupsv1.Host_Spec{
			DisplayName: "H2",
			Ips:         &commonpb.IPs{Ipv4: []string{"192.168.1.1"}},
			MetaInfo:    &sgroupsv1.Host_Spec_MetaInfo{Os: "windows"},
		},
	}

	got := HostFromProtoExt(ext)
	require.NotNil(t, got)
	require.Equal(t, []string{"192.168.1.1"}, got.IPs.IPv4)
	require.Nil(t, got.IPs.IPv6)
	require.Equal(t, "windows", got.MetaInfo.OS)
}

func TestHostFromProtoNoIPs(t *testing.T) {
	host := &sgroupsv1.Host{
		Metadata: &commonpb.Metadata{Name: "h3", Namespace: "default"},
		Spec:     &sgroupsv1.Host_Spec{DisplayName: "H3"},
	}

	got := HostFromProto(host)
	require.NotNil(t, got)
	require.Empty(t, got.IPs.IPv4)
	require.Empty(t, got.IPs.IPv6)
	require.Empty(t, got.MetaInfo.HostName)
}

func TestHostNilSafety(t *testing.T) {
	require.Nil(t, HostToProto(nil))
	require.Nil(t, HostFromProto(nil))
	require.Nil(t, HostFromProtoExt(nil))
}
