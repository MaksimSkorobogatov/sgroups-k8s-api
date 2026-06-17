package convert

import (
	"testing"

	agentv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/agent/v1"
	commonpb "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	"github.com/stretchr/testify/require"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestSocketStatListFromProto(t *testing.T) {
	in := []*sgroupsv1.HostResp_SocketStatistics_Host{
		{
			Name:      "host-1",
			Namespace: "default",
			Stats: []*agentv1.SockStat{
				{
					Protocol:  "tcp",
					Family:    commonpb.IpAddrFamily_IPV4,
					State:     agentv1.ConnState_LISTEN,
					LocalAddr: "0.0.0.0",
					LocalPort: 22,
					Inode:     1,
					Processes: []*agentv1.ProcessInfo{
						{Pid: 100, Comm: "sshd", Exe: "/usr/sbin/sshd", Fd: 3},
					},
				},
				nil, // must be skipped
				{
					Protocol:   "tcp",
					Family:     commonpb.IpAddrFamily_IPV6,
					State:      agentv1.ConnState_ESTABLISHED,
					LocalAddr:  "::1",
					LocalPort:  443,
					RemoteAddr: "::1",
					RemotePort: 51234,
					Inode:      2,
				},
			},
		},
	}

	out := SocketStatListFromProto(in)
	require.NotNil(t, out)
	require.Equal(t, v1alpha1.KindSocketStatList, out.Kind)
	require.Equal(t, v1alpha1.SchemeGroupVersion.String(), out.APIVersion)
	require.Equal(t, "host-1", out.Host.Name)
	require.Equal(t, "default", out.Host.Namespace)
	require.Len(t, out.Items, 2)

	require.Equal(t, "tcp", out.Items[0].Protocol)
	require.Equal(t, v1alpha1.IpAddrFamilyIPv4, out.Items[0].Family)
	require.Equal(t, v1alpha1.ConnStateListen, out.Items[0].State)
	require.Equal(t, int32(22), out.Items[0].LocalPort)
	require.Len(t, out.Items[0].Processes, 1)
	require.Equal(t, "sshd", out.Items[0].Processes[0].Comm)

	require.Equal(t, v1alpha1.IpAddrFamilyIPv6, out.Items[1].Family)
	require.Equal(t, v1alpha1.ConnStateEstablished, out.Items[1].State)
}

func TestSocketStatListFromProtoEmpty(t *testing.T) {
	out := SocketStatListFromProto(nil)
	require.NotNil(t, out)
	require.Empty(t, out.Items)
	require.Empty(t, out.Host.Name)
	require.Empty(t, out.Host.Namespace)
}

func TestSocketStatListFromProtoSingleEntry(t *testing.T) {
	// The subresource always queries one host; even though proto supports
	// multi-host responses, AGL only ever sees one entry. Anything beyond
	// hosts[0] is silently ignored (server-bug defence).
	in := []*sgroupsv1.HostResp_SocketStatistics_Host{
		{
			Name:  "expected",
			Stats: []*agentv1.SockStat{{Protocol: "tcp", LocalPort: 80}},
		},
		{
			Name:  "unexpected-extra",
			Stats: []*agentv1.SockStat{{Protocol: "udp", LocalPort: 53}},
		},
	}

	out := SocketStatListFromProto(in)
	require.Equal(t, "expected", out.Host.Name, "host attribution comes from hosts[0]")
	require.Len(t, out.Items, 1, "only the first host's stats should appear")
	require.Equal(t, int32(80), out.Items[0].LocalPort)
}

func TestSocketStatListFromProtoNilFirst(t *testing.T) {
	// Defensive: backend bug — first host is nil. We must not panic and the
	// result should be empty (no stats can be extracted from nothing).
	in := []*sgroupsv1.HostResp_SocketStatistics_Host{nil}
	out := SocketStatListFromProto(in)
	require.NotNil(t, out)
	require.Empty(t, out.Items)
}

func TestConnStateRoundTrip(t *testing.T) {
	tests := []struct {
		k8s   v1alpha1.ConnState
		proto agentv1.ConnState
	}{
		{v1alpha1.ConnStateEstablished, agentv1.ConnState_ESTABLISHED},
		{v1alpha1.ConnStateSynSent, agentv1.ConnState_SYN_SENT},
		{v1alpha1.ConnStateSynRecv, agentv1.ConnState_SYN_RECV},
		{v1alpha1.ConnStateFinWait1, agentv1.ConnState_FIN_WAIT1},
		{v1alpha1.ConnStateFinWait2, agentv1.ConnState_FIN_WAIT2},
		{v1alpha1.ConnStateTimeWait, agentv1.ConnState_TIME_WAIT},
		{v1alpha1.ConnStateClose, agentv1.ConnState_CLOSE},
		{v1alpha1.ConnStateCloseWait, agentv1.ConnState_CLOSE_WAIT},
		{v1alpha1.ConnStateLastAck, agentv1.ConnState_LAST_ACK},
		{v1alpha1.ConnStateListen, agentv1.ConnState_LISTEN},
		{v1alpha1.ConnStateClosing, agentv1.ConnState_CLOSING},
		{v1alpha1.ConnStateNewSynRecv, agentv1.ConnState_NEW_SYN_RECV},
	}
	for _, tt := range tests {
		t.Run(string(tt.k8s), func(t *testing.T) {
			require.Equal(t, tt.k8s, connStateFromProto(tt.proto))
			require.Equal(t, tt.proto, connStateToProto(tt.k8s))
		})
	}
}

func TestConnStateUnknownMapping(t *testing.T) {
	// proto-zero (CONN_UNDEF) and unknown-int both map to ConnStateUnknown.
	require.Equal(t, v1alpha1.ConnStateUnknown, connStateFromProto(agentv1.ConnState_CONN_UNDEF))
	require.Equal(t, v1alpha1.ConnStateUnknown, connStateFromProto(agentv1.ConnState(999)))
	// K8s-side "Unknown" and empty string both map to CONN_UNDEF.
	require.Equal(t, agentv1.ConnState_CONN_UNDEF, connStateToProto(v1alpha1.ConnStateUnknown))
	require.Equal(t, agentv1.ConnState_CONN_UNDEF, connStateToProto(""))
}

func TestSocketStatListRequest(t *testing.T) {
	req := SocketStatListRequest("default", "host-1", []v1alpha1.SocketStatSelector{
		{Protocol: "tcp", State: v1alpha1.ConnStateListen, LocalPort: 80},
	})
	require.NotNil(t, req)
	require.Len(t, req.Selectors, 1)
	require.Equal(t, "host-1", req.Selectors[0].Name)
	require.Equal(t, "default", req.Selectors[0].Namespace)
	require.Len(t, req.Selectors[0].Filters, 1)
	require.Equal(t, "tcp", req.Selectors[0].Filters[0].Protocol)
	require.Equal(t, agentv1.ConnState_LISTEN, req.Selectors[0].Filters[0].State)
	require.NotNil(t, req.Selectors[0].Filters[0].LocalPort)
	require.Equal(t, int32(80), *req.Selectors[0].Filters[0].LocalPort)
	// Unset (zero) selector fields → nil = "no constraint" on the wire.
	require.Nil(t, req.Selectors[0].Filters[0].RemotePort)
	require.Nil(t, req.Selectors[0].Filters[0].Inode)
	require.Nil(t, req.Selectors[0].Filters[0].Pid)
}

func TestSocketStatWatchRequest(t *testing.T) {
	req := SocketStatWatchRequest("ns", "h", nil)
	require.NotNil(t, req)
	require.Len(t, req.Selectors, 1)
	require.Equal(t, "h", req.Selectors[0].Name)
	require.Equal(t, "ns", req.Selectors[0].Namespace)
	require.Empty(t, req.Selectors[0].Filters)
}

// Sanity check that compile-time we still wire proto types correctly.
var _ = sgroupsv1.HostReq_SocketStatistics_List{}
var _ = sgroupsv1.HostResp_SocketStatistics_Watch{}
