package convert

import (
	agentv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/agent/v1"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// SocketStatListFromProto flattens hosts[0].stats into a K8s SocketStatList
// and surfaces the host name/namespace as top-level attribution.
// Extra hosts are ignored — subresource queries a single host by URL.
//
//nolint:dupl // parallel subresource list-converter; mirrors NftListFromProto by design.
func SocketStatListFromProto(in []*sgroupsv1.HostResp_SocketStatistics_Host) *v1alpha1.SocketStatList {
	out := &v1alpha1.SocketStatList{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.KindSocketStatList,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Items: []v1alpha1.SocketStat{},
	}
	if len(in) == 0 {
		return out
	}
	first := in[0]
	if first == nil {
		return out
	}
	out.Host = v1alpha1.ResourceIdentifier{
		Name:      first.GetName(),
		Namespace: first.GetNamespace(),
	}
	stats := first.GetStats()
	out.Items = make([]v1alpha1.SocketStat, 0, len(stats))
	for _, s := range stats {
		if s == nil {
			continue
		}
		out.Items = append(out.Items, socketStatFromProto(s))
	}

	return out
}

// SocketStatListRequest builds the gRPC List request for a single host.
// Backend treats {selectors:[]} as "match-all on that host".
func SocketStatListRequest(namespace, name string, selectors []v1alpha1.SocketStatSelector) *sgroupsv1.HostReq_SocketStatistics_List {
	return &sgroupsv1.HostReq_SocketStatistics_List{
		Selectors: []*sgroupsv1.HostReq_SocketStatistics_FieldSelector{
			{
				Name:      name,
				Namespace: namespace,
				Filters:   socketStatSelectorsToProto(selectors),
			},
		},
	}
}

// SocketStatWatchRequest mirrors SocketStatListRequest for the Watch RPC.
func SocketStatWatchRequest(namespace, name string, selectors []v1alpha1.SocketStatSelector) *sgroupsv1.HostReq_SocketStatistics_Watch {
	return &sgroupsv1.HostReq_SocketStatistics_Watch{
		Selectors: []*sgroupsv1.HostReq_SocketStatistics_FieldSelector{
			{
				Name:      name,
				Namespace: namespace,
				Filters:   socketStatSelectorsToProto(selectors),
			},
		},
	}
}

// socketStatFromProto converts a single agentv1.SockStat → v1alpha1.SocketStat.
func socketStatFromProto(in *agentv1.SockStat) v1alpha1.SocketStat {
	if in == nil {
		return v1alpha1.SocketStat{}
	}
	out := v1alpha1.SocketStat{
		Protocol:   in.GetProtocol(),
		Family:     ipAddrFamilyFromProto(in.GetFamily()),
		State:      connStateFromProto(in.GetState()),
		LocalAddr:  in.GetLocalAddr(),
		LocalPort:  in.GetLocalPort(),
		RemoteAddr: in.GetRemoteAddr(),
		RemotePort: in.GetRemotePort(),
		Ifname:     in.GetIfname(),
		Inode:      in.GetInode(),
	}
	if procs := in.GetProcesses(); len(procs) > 0 {
		out.Processes = make([]v1alpha1.Process, 0, len(procs))
		for _, p := range procs {
			if p == nil {
				continue
			}
			out.Processes = append(out.Processes, v1alpha1.Process{
				PID:     p.GetPid(),
				Comm:    p.GetComm(),
				CmdLine: p.GetCmdLine(),
				Exe:     p.GetExe(),
				FD:      p.GetFd(),
			})
		}
	}

	return out
}

func socketStatSelectorsToProto(in []v1alpha1.SocketStatSelector) []*agentv1.SockStat_Selectors {
	if len(in) == 0 {
		return nil
	}
	out := make([]*agentv1.SockStat_Selectors, 0, len(in))
	for _, s := range in {
		out = append(out, &agentv1.SockStat_Selectors{
			Protocol:   s.Protocol,
			Family:     ipAddrFamilyToProto(s.Family),
			State:      connStateToProto(s.State),
			LocalAddr:  s.LocalAddr,
			LocalPort:  optInt32(s.LocalPort),
			RemoteAddr: s.RemoteAddr,
			RemotePort: optInt32(s.RemotePort),
			Ifname:     s.Ifname,
			Inode:      optInt64(s.Inode),
			Pid:        optInt32(s.PID),
			Comm:       s.Comm,
		})
	}

	return out
}

// optInt32/optInt64 wrap a selector field: zero means "no constraint" (matches
// the k8s value-type + omitempty contract) → nil on the wire, so the backend
// distinguishes "no filter" from "match exactly 0".
func optInt32(v int32) *int32 {
	if v == 0 {
		return nil
	}

	return &v
}

func optInt64(v int64) *int64 {
	if v == 0 {
		return nil
	}

	return &v
}

// connStateProtoToK8s is the source of truth for ConnState mapping; the
// inverse table is derived from it to prevent drift.
var connStateProtoToK8s = map[agentv1.ConnState]v1alpha1.ConnState{
	agentv1.ConnState_ESTABLISHED:  v1alpha1.ConnStateEstablished,
	agentv1.ConnState_SYN_SENT:     v1alpha1.ConnStateSynSent,
	agentv1.ConnState_SYN_RECV:     v1alpha1.ConnStateSynRecv,
	agentv1.ConnState_FIN_WAIT1:    v1alpha1.ConnStateFinWait1,
	agentv1.ConnState_FIN_WAIT2:    v1alpha1.ConnStateFinWait2,
	agentv1.ConnState_TIME_WAIT:    v1alpha1.ConnStateTimeWait,
	agentv1.ConnState_CLOSE:        v1alpha1.ConnStateClose,
	agentv1.ConnState_CLOSE_WAIT:   v1alpha1.ConnStateCloseWait,
	agentv1.ConnState_LAST_ACK:     v1alpha1.ConnStateLastAck,
	agentv1.ConnState_LISTEN:       v1alpha1.ConnStateListen,
	agentv1.ConnState_CLOSING:      v1alpha1.ConnStateClosing,
	agentv1.ConnState_NEW_SYN_RECV: v1alpha1.ConnStateNewSynRecv,
}

var connStateK8sToProto = func() map[v1alpha1.ConnState]agentv1.ConnState {
	out := make(map[v1alpha1.ConnState]agentv1.ConnState, len(connStateProtoToK8s))
	for p, k := range connStateProtoToK8s {
		out[k] = p
	}

	return out
}()

// connStateFromProto maps proto to K8s; unknown values become ConnStateUnknown
// (passthrough — surfacing data should not error on a new state).
func connStateFromProto(s agentv1.ConnState) v1alpha1.ConnState {
	if v, ok := connStateProtoToK8s[s]; ok {
		return v
	}

	return v1alpha1.ConnStateUnknown
}

// connStateToProto maps K8s to proto; empty / Unknown become CONN_UNDEF so the
// backend treats "no constraint" the same as wire-default.
func connStateToProto(s v1alpha1.ConnState) agentv1.ConnState {
	if v, ok := connStateK8sToProto[s]; ok {
		return v
	}

	return agentv1.ConnState_CONN_UNDEF
}
