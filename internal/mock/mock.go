package mock

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"sgroups.io/sgroups-k8s-api/internal/backend"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"

	agentv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/agent/v1"
	commonpb "github.com/PRO-Robotech/sgroups-proto/pkg/api/common"
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
)

type MockBackend struct {
	mu              sync.RWMutex
	version         atomic.Int64
	namespaces      map[string]*sgroupsv1.Namespace
	addressGroups   map[string]*sgroupsv1.AddressGroup
	networks        map[string]*sgroupsv1.Network
	hosts           map[string]*sgroupsv1.Host
	hostBindings    map[string]*sgroupsv1.HostBinding
	networkBindings map[string]*sgroupsv1.NetworkBinding
	services        map[string]*sgroupsv1.Service
	serviceBindings map[string]*sgroupsv1.ServiceBinding
	rules           map[string]*sgroupsv1.Rule
	nsHub           *namespaceWatchHub
	agHub           *addressGroupWatchHub
	nwHub           *networkWatchHub
	hostHub         *hostWatchHub
	hbHub           *hostBindingWatchHub
	nbHub           *networkBindingWatchHub
	svcHub          *serviceWatchHub
	sbHub           *serviceBindingWatchHub
	ruleHub         *ruleWatchHub
}

func New() *MockBackend {
	return &MockBackend{
		namespaces:      make(map[string]*sgroupsv1.Namespace),
		addressGroups:   make(map[string]*sgroupsv1.AddressGroup),
		networks:        make(map[string]*sgroupsv1.Network),
		hosts:           make(map[string]*sgroupsv1.Host),
		hostBindings:    make(map[string]*sgroupsv1.HostBinding),
		networkBindings: make(map[string]*sgroupsv1.NetworkBinding),
		services:        make(map[string]*sgroupsv1.Service),
		serviceBindings: make(map[string]*sgroupsv1.ServiceBinding),
		rules:           make(map[string]*sgroupsv1.Rule),
		nsHub:           newNamespaceWatchHub(),
		agHub:           newAddressGroupWatchHub(),
		nwHub:           newNetworkWatchHub(),
		hostHub:         newHostWatchHub(),
		hbHub:           newHostBindingWatchHub(),
		nbHub:           newNetworkBindingWatchHub(),
		svcHub:          newServiceWatchHub(),
		sbHub:           newServiceBindingWatchHub(),
		ruleHub:         newRuleWatchHub(),
	}
}

func (m *MockBackend) UpsertNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Upsert) (*sgroupsv1.NamespaceResp_Upsert, error) {
	if req == nil || len(req.Namespaces) == 0 {
		return nil, errors.New("namespaces are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.Namespace, 0, len(req.Namespaces))
	modified := make([]*sgroupsv1.Namespace, 0, len(req.Namespaces))
	resp := &sgroupsv1.NamespaceResp_Upsert{Namespaces: make([]*sgroupsv1.Namespace, 0, len(req.Namespaces))}
	for _, ns := range req.Namespaces {
		if ns == nil || ns.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("namespace metadata is required")
		}
		stored, isNew := m.upsertNamespaceLocked(ns)
		resp.Namespaces = append(resp.Namespaces, cloneNamespace(stored))
		if isNew {
			added = append(added, cloneNamespace(stored))
		} else {
			modified = append(modified, cloneNamespace(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.nsHub.publish(&sgroupsv1.NamespaceResp_Watch{Type: commonpb.WatchEventType_ADDED, Namespaces: added})
	}
	if len(modified) > 0 {
		m.nsHub.publish(&sgroupsv1.NamespaceResp_Watch{Type: commonpb.WatchEventType_MODIFIED, Namespaces: modified})
	}

	return resp, nil
}

func (m *MockBackend) DeleteNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Delete) error {
	if req == nil || len(req.Namespaces) == 0 {
		return errors.New("namespaces are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.Namespace, 0, len(req.Namespaces))
	for _, ns := range req.Namespaces {
		if ns == nil || ns.Metadata == nil {
			m.mu.Unlock()

			return errors.New("namespace delete metadata is required")
		}
		uid := ns.Metadata.Uid
		name := ns.Metadata.Name
		if uid == "" && name == "" {
			m.mu.Unlock()

			return errors.New("namespace delete requires uid or name")
		}
		if uid != "" {
			if stored, ok := m.namespaces[uid]; ok {
				delete(m.namespaces, uid)
				deleted = append(deleted, cloneNamespace(stored))
			}

			continue
		}
		for id, stored := range m.namespaces {
			if stored.Metadata != nil && stored.Metadata.Name == name {
				delete(m.namespaces, id)
				deleted = append(deleted, cloneNamespace(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.nsHub.publish(&sgroupsv1.NamespaceResp_Watch{Type: commonpb.WatchEventType_DELETED, Namespaces: deleted})
	}

	return nil
}

func (m *MockBackend) ListNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_List) (*sgroupsv1.NamespaceResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.Namespace, 0, len(m.namespaces))
	for _, ns := range m.namespaces {
		items = append(items, ns)
	}
	m.mu.RUnlock()

	filtered := filterNamespaces(req.Selectors, items)

	return &sgroupsv1.NamespaceResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		Namespaces:      filtered,
	}, nil
}

func (m *MockBackend) WatchNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Watch) (backend.WatchStream[*sgroupsv1.NamespaceResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.NamespaceResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.nsHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.Namespace, 0, len(m.namespaces))
	for _, ns := range m.namespaces {
		items = append(items, ns)
	}
	m.mu.RUnlock()

	snapshot := filterNamespaces(req.Selectors, items)
	ch <- &sgroupsv1.NamespaceResp_Watch{Type: commonpb.WatchEventType_ADDED, Namespaces: snapshot}

	return backend.WatchStream[*sgroupsv1.NamespaceResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) UpsertAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Upsert) (*sgroupsv1.AddressGroupResp_Upsert, error) {
	if req == nil || len(req.AddressGroups) == 0 {
		return nil, errors.New("address groups are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.AddressGroup, 0, len(req.AddressGroups))
	modified := make([]*sgroupsv1.AddressGroup, 0, len(req.AddressGroups))
	resp := &sgroupsv1.AddressGroupResp_Upsert{AddressGroups: make([]*sgroupsv1.AddressGroup, 0, len(req.AddressGroups))}
	for _, ag := range req.AddressGroups {
		if ag == nil || ag.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("address group metadata is required")
		}
		stored, isNew := m.upsertAddressGroupLocked(ag)
		resp.AddressGroups = append(resp.AddressGroups, cloneAddressGroup(stored))
		if isNew {
			added = append(added, cloneAddressGroup(stored))
		} else {
			modified = append(modified, cloneAddressGroup(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.agHub.publish(&sgroupsv1.AddressGroupResp_Watch{Type: commonpb.WatchEventType_ADDED, AddressGroups: toAddressGroupExtList(added)})
	}
	if len(modified) > 0 {
		m.agHub.publish(&sgroupsv1.AddressGroupResp_Watch{Type: commonpb.WatchEventType_MODIFIED, AddressGroups: toAddressGroupExtList(modified)})
	}

	return resp, nil
}

func (m *MockBackend) DeleteAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Delete) error {
	if req == nil || len(req.AddressGroups) == 0 {
		return errors.New("address groups are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.AddressGroup, 0, len(req.AddressGroups))
	for _, ag := range req.AddressGroups {
		if ag == nil || ag.Metadata == nil {
			m.mu.Unlock()

			return errors.New("address group delete metadata is required")
		}
		uid := ag.Metadata.Uid
		name := ag.Metadata.Name
		ns := ag.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("address group delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.addressGroups[uid]; ok {
				delete(m.addressGroups, uid)
				deleted = append(deleted, cloneAddressGroup(stored))
			}

			continue
		}
		for id, stored := range m.addressGroups {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.addressGroups, id)
				deleted = append(deleted, cloneAddressGroup(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.agHub.publish(&sgroupsv1.AddressGroupResp_Watch{Type: commonpb.WatchEventType_DELETED, AddressGroups: toAddressGroupExtList(deleted)})
	}

	return nil
}

func (m *MockBackend) ListAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_List) (*sgroupsv1.AddressGroupResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.AddressGroup, 0, len(m.addressGroups))
	for _, ag := range m.addressGroups {
		items = append(items, ag)
	}

	filtered := filterAddressGroups(req.Selectors, items)
	extList := toAddressGroupExtList(filtered)
	m.fillAddressGroupRefs(extList)
	m.mu.RUnlock()

	return &sgroupsv1.AddressGroupResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		AddressGroups:   extList,
	}, nil
}

func (m *MockBackend) WatchAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Watch) (backend.WatchStream[*sgroupsv1.AddressGroupResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.AddressGroupResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.agHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.AddressGroup, 0, len(m.addressGroups))
	for _, ag := range m.addressGroups {
		items = append(items, ag)
	}
	m.mu.RUnlock()

	snapshot := filterAddressGroups(req.Selectors, items)
	ch <- &sgroupsv1.AddressGroupResp_Watch{Type: commonpb.WatchEventType_ADDED, AddressGroups: toAddressGroupExtList(snapshot)}

	return backend.WatchStream[*sgroupsv1.AddressGroupResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) UpsertNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Upsert) (*sgroupsv1.NetworkResp_Upsert, error) {
	if req == nil || len(req.Networks) == 0 {
		return nil, errors.New("networks are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.Network, 0, len(req.Networks))
	modified := make([]*sgroupsv1.Network, 0, len(req.Networks))
	resp := &sgroupsv1.NetworkResp_Upsert{Networks: make([]*sgroupsv1.Network, 0, len(req.Networks))}
	for _, nw := range req.Networks {
		if nw == nil || nw.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("network metadata is required")
		}
		stored, isNew := m.upsertNetworkLocked(nw)
		resp.Networks = append(resp.Networks, cloneNetwork(stored))
		if isNew {
			added = append(added, cloneNetwork(stored))
		} else {
			modified = append(modified, cloneNetwork(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.nwHub.publish(&sgroupsv1.NetworkResp_Watch{Type: commonpb.WatchEventType_ADDED, Networks: toNetworkExtList(added)})
	}
	if len(modified) > 0 {
		m.nwHub.publish(&sgroupsv1.NetworkResp_Watch{Type: commonpb.WatchEventType_MODIFIED, Networks: toNetworkExtList(modified)})
	}

	return resp, nil
}

func (m *MockBackend) DeleteNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Delete) error {
	if req == nil || len(req.Networks) == 0 {
		return errors.New("networks are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.Network, 0, len(req.Networks))
	for _, nw := range req.Networks {
		if nw == nil || nw.Metadata == nil {
			m.mu.Unlock()

			return errors.New("network delete metadata is required")
		}
		uid := nw.Metadata.Uid
		name := nw.Metadata.Name
		ns := nw.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("network delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.networks[uid]; ok {
				delete(m.networks, uid)
				deleted = append(deleted, cloneNetwork(stored))
			}

			continue
		}
		for id, stored := range m.networks {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.networks, id)
				deleted = append(deleted, cloneNetwork(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.nwHub.publish(&sgroupsv1.NetworkResp_Watch{Type: commonpb.WatchEventType_DELETED, Networks: toNetworkExtList(deleted)})
	}

	return nil
}

func (m *MockBackend) ListNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_List) (*sgroupsv1.NetworkResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.Network, 0, len(m.networks))
	for _, nw := range m.networks {
		items = append(items, nw)
	}

	filtered := filterNetworks(req.Selectors, items)
	extList := toNetworkExtList(filtered)
	m.fillNetworkRefs(extList)
	m.mu.RUnlock()

	return &sgroupsv1.NetworkResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		Networks:        extList,
	}, nil
}

func (m *MockBackend) WatchNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Watch) (backend.WatchStream[*sgroupsv1.NetworkResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.NetworkResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.nwHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.Network, 0, len(m.networks))
	for _, nw := range m.networks {
		items = append(items, nw)
	}
	m.mu.RUnlock()

	snapshot := filterNetworks(req.Selectors, items)
	ch <- &sgroupsv1.NetworkResp_Watch{Type: commonpb.WatchEventType_ADDED, Networks: toNetworkExtList(snapshot)}

	return backend.WatchStream[*sgroupsv1.NetworkResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) UpsertHosts(ctx context.Context, req *sgroupsv1.HostReq_Upsert) (*sgroupsv1.HostResp_Upsert, error) {
	if req == nil || len(req.Hosts) == 0 {
		return nil, errors.New("hosts are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.Host, 0, len(req.Hosts))
	modified := make([]*sgroupsv1.Host, 0, len(req.Hosts))
	resp := &sgroupsv1.HostResp_Upsert{Hosts: make([]*sgroupsv1.Host, 0, len(req.Hosts))}
	for _, h := range req.Hosts {
		if h == nil || h.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("host metadata is required")
		}
		stored, isNew := m.upsertHostLocked(h)
		resp.Hosts = append(resp.Hosts, cloneHost(stored))
		if isNew {
			added = append(added, cloneHost(stored))
		} else {
			modified = append(modified, cloneHost(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.hostHub.publish(&sgroupsv1.HostResp_Watch{Type: commonpb.WatchEventType_ADDED, Hosts: toHostExtList(added)})
	}
	if len(modified) > 0 {
		m.hostHub.publish(&sgroupsv1.HostResp_Watch{Type: commonpb.WatchEventType_MODIFIED, Hosts: toHostExtList(modified)})
	}

	return resp, nil
}

func (m *MockBackend) DeleteHosts(ctx context.Context, req *sgroupsv1.HostReq_Delete) error {
	if req == nil || len(req.Hosts) == 0 {
		return errors.New("hosts are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.Host, 0, len(req.Hosts))
	for _, h := range req.Hosts {
		if h == nil || h.Metadata == nil {
			m.mu.Unlock()

			return errors.New("host delete metadata is required")
		}
		uid := h.Metadata.Uid
		name := h.Metadata.Name
		ns := h.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("host delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.hosts[uid]; ok {
				delete(m.hosts, uid)
				deleted = append(deleted, cloneHost(stored))
			}

			continue
		}
		for id, stored := range m.hosts {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.hosts, id)
				deleted = append(deleted, cloneHost(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.hostHub.publish(&sgroupsv1.HostResp_Watch{Type: commonpb.WatchEventType_DELETED, Hosts: toHostExtList(deleted)})
	}

	return nil
}

func (m *MockBackend) ListHosts(ctx context.Context, req *sgroupsv1.HostReq_List) (*sgroupsv1.HostResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.Host, 0, len(m.hosts))
	for _, h := range m.hosts {
		items = append(items, h)
	}

	filtered := filterHosts(req.Selectors, items)
	extList := toHostExtList(filtered)
	m.fillHostRefs(extList)
	m.mu.RUnlock()

	return &sgroupsv1.HostResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		Hosts:           extList,
	}, nil
}

func (m *MockBackend) WatchHosts(ctx context.Context, req *sgroupsv1.HostReq_Watch) (backend.WatchStream[*sgroupsv1.HostResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.HostResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.hostHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.Host, 0, len(m.hosts))
	for _, h := range m.hosts {
		items = append(items, h)
	}
	m.mu.RUnlock()

	snapshot := filterHosts(req.Selectors, items)
	ch <- &sgroupsv1.HostResp_Watch{Type: commonpb.WatchEventType_ADDED, Hosts: toHostExtList(snapshot)}

	return backend.WatchStream[*sgroupsv1.HostResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

// ListSocketStatistics returns fabricated stats per selected host so the
// subresource has something to surface end-to-end.
func (m *MockBackend) ListSocketStatistics(ctx context.Context, req *sgroupsv1.HostReq_SocketStatistics_List) (*sgroupsv1.HostResp_SocketStatistics_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}

	return &sgroupsv1.HostResp_SocketStatistics_List{
		Hosts: m.fabricatedSocketStatsHosts(req.GetSelectors()),
	}, nil
}

// WatchSocketStatistics emits one snapshot and then idles — enough to
// exercise the streaming path in tests.
func (m *MockBackend) WatchSocketStatistics(ctx context.Context, req *sgroupsv1.HostReq_SocketStatistics_Watch) (backend.WatchStream[*sgroupsv1.HostResp_SocketStatistics_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.HostResp_SocketStatistics_Watch]{}, errors.New("selectors are required")
	}

	ch := make(chan *sgroupsv1.HostResp_SocketStatistics_Watch, 1)
	ch <- &sgroupsv1.HostResp_SocketStatistics_Watch{
		Hosts: m.fabricatedSocketStatsHosts(req.GetSelectors()),
	}

	return backend.WatchStream[*sgroupsv1.HostResp_SocketStatistics_Watch]{
		C:     ch,
		Close: func() { close(ch) },
	}, nil
}

// ListNft returns a fabricated ruleset per selected host so the subresource
// has something to surface end-to-end.
func (m *MockBackend) ListNft(ctx context.Context, req *sgroupsv1.HostReq_Nft_List) (*sgroupsv1.HostResp_Nft_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}

	return &sgroupsv1.HostResp_Nft_List{
		Hosts: m.fabricatedNftHosts(req.GetSelectors()),
	}, nil
}

// WatchNft emits one snapshot and then idles — enough to exercise the
// streaming path in tests.
func (m *MockBackend) WatchNft(ctx context.Context, req *sgroupsv1.HostReq_Nft_Watch) (backend.WatchStream[*sgroupsv1.HostResp_Nft_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.HostResp_Nft_Watch]{}, errors.New("selectors are required")
	}

	ch := make(chan *sgroupsv1.HostResp_Nft_Watch, 1)
	ch <- &sgroupsv1.HostResp_Nft_Watch{
		Hosts: m.fabricatedNftHosts(req.GetSelectors()),
	}

	return backend.WatchStream[*sgroupsv1.HostResp_Nft_Watch]{
		C:     ch,
		Close: func() { close(ch) },
	}, nil
}

// fabricatedNftHosts builds a Host-wrapped ruleset per selector. Selectors
// pointing at unknown hosts are silently dropped (matching how the real
// backend handles fan-out to an unreachable agent).
func (m *MockBackend) fabricatedNftHosts(selectors []*sgroupsv1.HostReq_Nft_FieldSelector) []*sgroupsv1.HostResp_Nft_Host {
	out := make([]*sgroupsv1.HostResp_Nft_Host, 0, len(selectors))
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if !m.hostExists(sel.GetName(), sel.GetNamespace()) {
			continue
		}
		out = append(out, &sgroupsv1.HostResp_Nft_Host{
			Name:      sel.GetName(),
			Namespace: sel.GetNamespace(),
			Nft:       fabricateNftables(),
		})
	}

	return out
}

// fabricateNftables returns a deterministic single-ruleset dataset in both
// text and json form, so the subresource exercises the structured-JSON path.
func fabricateNftables() []*agentv1.Nftables {
	return []*agentv1.Nftables{
		{
			Text: "table inet filter {\n\tchain input {\n\t\ttype filter hook input priority 0; policy drop;\n\t}\n}\n",
			Json: `{"nftables":[{"table":{"family":"inet","name":"filter","handle":1}}]}`,
		},
	}
}

// fabricatedSocketStatsHosts builds a Host-wrapped response per selector.
// Selectors pointing at unknown hosts are silently dropped (matching how the
// real backend handles fan-out to an unreachable agent).
func (m *MockBackend) fabricatedSocketStatsHosts(selectors []*sgroupsv1.HostReq_SocketStatistics_FieldSelector) []*sgroupsv1.HostResp_SocketStatistics_Host {
	out := make([]*sgroupsv1.HostResp_SocketStatistics_Host, 0, len(selectors))
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if !m.hostExists(sel.GetName(), sel.GetNamespace()) {
			continue
		}
		out = append(out, &sgroupsv1.HostResp_SocketStatistics_Host{
			Name:      sel.GetName(),
			Namespace: sel.GetNamespace(),
			Stats:     fabricateSockStats(sel.GetFilters()),
		})
	}

	return out
}

// hostExists checks whether a host (name@namespace) is in storage.
func (m *MockBackend) hostExists(name, namespace string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, h := range m.hosts {
		md := h.GetMetadata()
		if md.GetName() == name && md.GetNamespace() == namespace {
			return true
		}
	}

	return false
}

// fabricateSockStats returns a deterministic dataset; filters, if any, drop
// non-matching entries (mock-grade: Protocol / State / Port only).
func fabricateSockStats(filters []*agentv1.SockStat_Selectors) []*agentv1.SockStat {
	all := []*agentv1.SockStat{
		{
			Protocol:  "tcp",
			Family:    commonpb.IpAddrFamily_IPV4,
			State:     agentv1.ConnState_LISTEN,
			LocalAddr: "0.0.0.0",
			LocalPort: 22,
			Inode:     1,
			Processes: []*agentv1.ProcessInfo{{Pid: 100, Comm: "sshd", Exe: "/usr/sbin/sshd", Fd: 3}},
		},
		{
			Protocol:  "tcp",
			Family:    commonpb.IpAddrFamily_IPV4,
			State:     agentv1.ConnState_LISTEN,
			LocalAddr: "0.0.0.0",
			LocalPort: 80,
			Inode:     2,
			Processes: []*agentv1.ProcessInfo{{Pid: 200, Comm: "nginx", Exe: "/usr/sbin/nginx", Fd: 6}},
		},
		{
			Protocol:  "tcp",
			Family:    commonpb.IpAddrFamily_IPV4,
			State:     agentv1.ConnState_LISTEN,
			LocalAddr: "0.0.0.0",
			LocalPort: 443,
			Inode:     3,
			Processes: []*agentv1.ProcessInfo{{Pid: 200, Comm: "nginx", Exe: "/usr/sbin/nginx", Fd: 7}},
		},
		{
			Protocol:   "tcp",
			Family:     commonpb.IpAddrFamily_IPV4,
			State:      agentv1.ConnState_ESTABLISHED,
			LocalAddr:  "10.0.0.1",
			LocalPort:  443,
			RemoteAddr: "10.0.0.42",
			RemotePort: 51234,
			Inode:      4,
			Processes:  []*agentv1.ProcessInfo{{Pid: 200, Comm: "nginx", Exe: "/usr/sbin/nginx", Fd: 12}},
		},
	}

	if len(filters) == 0 {
		return all
	}
	out := make([]*agentv1.SockStat, 0, len(all))
	for _, s := range all {
		if matchSockStat(s, filters) {
			out = append(out, s)
		}
	}

	return out
}

// matchSockStat returns true if any filter matches. A field on the filter is
// honored only when set; multiple set fields are AND-joined within one filter.
func matchSockStat(s *agentv1.SockStat, filters []*agentv1.SockStat_Selectors) bool {
	for _, f := range filters {
		if f == nil {
			continue
		}
		if f.Protocol != "" && f.Protocol != s.Protocol {
			continue
		}
		if f.State != agentv1.ConnState_CONN_UNDEF && f.State != s.State {
			continue
		}
		if f.LocalPort != nil && *f.LocalPort != s.LocalPort {
			continue
		}
		if f.RemotePort != nil && *f.RemotePort != s.RemotePort {
			continue
		}

		return true
	}

	return false
}

func (m *MockBackend) UpsertHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Upsert) (*sgroupsv1.HostBindingResp_Upsert, error) {
	if req == nil || len(req.HostBindings) == 0 {
		return nil, errors.New("host bindings are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.HostBinding, 0, len(req.HostBindings))
	modified := make([]*sgroupsv1.HostBinding, 0, len(req.HostBindings))
	resp := &sgroupsv1.HostBindingResp_Upsert{HostBindings: make([]*sgroupsv1.HostBinding, 0, len(req.HostBindings))}
	for _, hb := range req.HostBindings {
		if hb == nil || hb.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("host binding metadata is required")
		}
		stored, isNew := m.upsertHostBindingLocked(hb)
		resp.HostBindings = append(resp.HostBindings, cloneHostBinding(stored))
		if isNew {
			added = append(added, cloneHostBinding(stored))
		} else {
			modified = append(modified, cloneHostBinding(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.hbHub.publish(&sgroupsv1.HostBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, HostBindings: added})
	}
	if len(modified) > 0 {
		m.hbHub.publish(&sgroupsv1.HostBindingResp_Watch{Type: commonpb.WatchEventType_MODIFIED, HostBindings: modified})
	}

	return resp, nil
}

func (m *MockBackend) DeleteHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Delete) error {
	if req == nil || len(req.HostBindings) == 0 {
		return errors.New("host bindings are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.HostBinding, 0, len(req.HostBindings))
	for _, hb := range req.HostBindings {
		if hb == nil || hb.Metadata == nil {
			m.mu.Unlock()

			return errors.New("host binding delete metadata is required")
		}
		uid := hb.Metadata.Uid
		name := hb.Metadata.Name
		ns := hb.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("host binding delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.hostBindings[uid]; ok {
				delete(m.hostBindings, uid)
				deleted = append(deleted, cloneHostBinding(stored))
			}

			continue
		}
		for id, stored := range m.hostBindings {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.hostBindings, id)
				deleted = append(deleted, cloneHostBinding(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.hbHub.publish(&sgroupsv1.HostBindingResp_Watch{Type: commonpb.WatchEventType_DELETED, HostBindings: deleted})
	}

	return nil
}

func (m *MockBackend) ListHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_List) (*sgroupsv1.HostBindingResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.HostBinding, 0, len(m.hostBindings))
	for _, hb := range m.hostBindings {
		items = append(items, hb)
	}
	m.mu.RUnlock()

	filtered := filterHostBindings(req.Selectors, items)

	return &sgroupsv1.HostBindingResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		HostBindings:    filtered,
	}, nil
}

func (m *MockBackend) WatchHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Watch) (backend.WatchStream[*sgroupsv1.HostBindingResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.HostBindingResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.hbHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.HostBinding, 0, len(m.hostBindings))
	for _, hb := range m.hostBindings {
		items = append(items, hb)
	}
	m.mu.RUnlock()

	snapshot := filterHostBindings(req.Selectors, items)
	ch <- &sgroupsv1.HostBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, HostBindings: snapshot}

	return backend.WatchStream[*sgroupsv1.HostBindingResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) UpsertNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Upsert) (*sgroupsv1.NetworkBindingResp_Upsert, error) {
	if req == nil || len(req.NetworkBindings) == 0 {
		return nil, errors.New("network bindings are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.NetworkBinding, 0, len(req.NetworkBindings))
	modified := make([]*sgroupsv1.NetworkBinding, 0, len(req.NetworkBindings))
	resp := &sgroupsv1.NetworkBindingResp_Upsert{NetworkBindings: make([]*sgroupsv1.NetworkBinding, 0, len(req.NetworkBindings))}
	for _, nb := range req.NetworkBindings {
		if nb == nil || nb.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("network binding metadata is required")
		}
		stored, isNew := m.upsertNetworkBindingLocked(nb)
		resp.NetworkBindings = append(resp.NetworkBindings, cloneNetworkBinding(stored))
		if isNew {
			added = append(added, cloneNetworkBinding(stored))
		} else {
			modified = append(modified, cloneNetworkBinding(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.nbHub.publish(&sgroupsv1.NetworkBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, NetworkBindings: added})
	}
	if len(modified) > 0 {
		m.nbHub.publish(&sgroupsv1.NetworkBindingResp_Watch{Type: commonpb.WatchEventType_MODIFIED, NetworkBindings: modified})
	}

	return resp, nil
}

func (m *MockBackend) DeleteNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Delete) error {
	if req == nil || len(req.NetworkBindings) == 0 {
		return errors.New("network bindings are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.NetworkBinding, 0, len(req.NetworkBindings))
	for _, nb := range req.NetworkBindings {
		if nb == nil || nb.Metadata == nil {
			m.mu.Unlock()

			return errors.New("network binding delete metadata is required")
		}
		uid := nb.Metadata.Uid
		name := nb.Metadata.Name
		ns := nb.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("network binding delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.networkBindings[uid]; ok {
				delete(m.networkBindings, uid)
				deleted = append(deleted, cloneNetworkBinding(stored))
			}

			continue
		}
		for id, stored := range m.networkBindings {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.networkBindings, id)
				deleted = append(deleted, cloneNetworkBinding(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.nbHub.publish(&sgroupsv1.NetworkBindingResp_Watch{Type: commonpb.WatchEventType_DELETED, NetworkBindings: deleted})
	}

	return nil
}

func (m *MockBackend) ListNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_List) (*sgroupsv1.NetworkBindingResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.NetworkBinding, 0, len(m.networkBindings))
	for _, nb := range m.networkBindings {
		items = append(items, nb)
	}
	m.mu.RUnlock()

	filtered := filterNetworkBindings(req.Selectors, items)

	return &sgroupsv1.NetworkBindingResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		NetworkBindings: filtered,
	}, nil
}

func (m *MockBackend) WatchNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Watch) (backend.WatchStream[*sgroupsv1.NetworkBindingResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.NetworkBindingResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.nbHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.NetworkBinding, 0, len(m.networkBindings))
	for _, nb := range m.networkBindings {
		items = append(items, nb)
	}
	m.mu.RUnlock()

	snapshot := filterNetworkBindings(req.Selectors, items)
	ch <- &sgroupsv1.NetworkBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, NetworkBindings: snapshot}

	return backend.WatchStream[*sgroupsv1.NetworkBindingResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) upsertNamespaceLocked(ns *sgroupsv1.Namespace) (*sgroupsv1.Namespace, bool) {
	clone := cloneNamespace(ns)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.namespaces[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.namespaces[uid] = clone

	return clone, !ok
}

func (m *MockBackend) upsertAddressGroupLocked(ag *sgroupsv1.AddressGroup) (*sgroupsv1.AddressGroup, bool) {
	clone := cloneAddressGroup(ag)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.addressGroups[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.addressGroups[uid] = clone

	return clone, !ok
}

func (m *MockBackend) upsertHostLocked(h *sgroupsv1.Host) (*sgroupsv1.Host, bool) {
	clone := cloneHost(h)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.hosts[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.hosts[uid] = clone

	return clone, !ok
}

func (m *MockBackend) upsertHostBindingLocked(hb *sgroupsv1.HostBinding) (*sgroupsv1.HostBinding, bool) {
	clone := cloneHostBinding(hb)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.hostBindings[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.hostBindings[uid] = clone

	return clone, !ok
}

func (m *MockBackend) upsertNetworkBindingLocked(nb *sgroupsv1.NetworkBinding) (*sgroupsv1.NetworkBinding, bool) {
	clone := cloneNetworkBinding(nb)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.networkBindings[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.networkBindings[uid] = clone

	return clone, !ok
}

func (m *MockBackend) upsertNetworkLocked(nw *sgroupsv1.Network) (*sgroupsv1.Network, bool) {
	clone := cloneNetwork(nw)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.networks[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.networks[uid] = clone

	return clone, !ok
}

func cloneNamespace(ns *sgroupsv1.Namespace) *sgroupsv1.Namespace {
	if ns == nil {
		return nil
	}

	return proto.Clone(ns).(*sgroupsv1.Namespace) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func cloneAddressGroup(ag *sgroupsv1.AddressGroup) *sgroupsv1.AddressGroup {
	if ag == nil {
		return nil
	}

	return proto.Clone(ag).(*sgroupsv1.AddressGroup) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func toAddressGroupExtList(items []*sgroupsv1.AddressGroup) []*sgroupsv1.AddressGroupResp_AddressGroupExt {
	out := make([]*sgroupsv1.AddressGroupResp_AddressGroupExt, 0, len(items))
	for _, ag := range items {
		if ag == nil {
			continue
		}
		out = append(out, &sgroupsv1.AddressGroupResp_AddressGroupExt{
			Metadata: ag.GetMetadata(),
			Spec:     ag.GetSpec(),
		})
	}

	return out
}

// fillAddressGroupRefs computes refs for each AddressGroup by scanning bindings.
// Must be called while m.mu.RLock is held.
func (m *MockBackend) fillAddressGroupRefs(extList []*sgroupsv1.AddressGroupResp_AddressGroupExt) {
	for _, ext := range extList {
		name := ext.GetMetadata().GetName()
		ns := ext.GetMetadata().GetNamespace()
		var refs []*commonpb.ResourceRef
		for _, hb := range m.hostBindings {
			ag := hb.GetSpec().GetAddressGroup()
			if ag.GetName() == name && ag.GetNamespace() == ns {
				h := hb.GetSpec().GetHost()
				refs = append(refs, &commonpb.ResourceRef{Name: h.GetName(), Namespace: h.GetNamespace(), ResType: v1alpha1.KindHost})
			}
		}
		for _, nb := range m.networkBindings {
			ag := nb.GetSpec().GetAddressGroup()
			if ag.GetName() == name && ag.GetNamespace() == ns {
				nw := nb.GetSpec().GetNetwork()
				refs = append(refs, &commonpb.ResourceRef{Name: nw.GetName(), Namespace: nw.GetNamespace(), ResType: v1alpha1.KindNetwork})
			}
		}
		for _, sb := range m.serviceBindings {
			ag := sb.GetSpec().GetAddressGroup()
			if ag.GetName() == name && ag.GetNamespace() == ns {
				svc := sb.GetSpec().GetService()
				refs = append(refs, &commonpb.ResourceRef{Name: svc.GetName(), Namespace: svc.GetNamespace(), ResType: v1alpha1.KindService})
			}
		}
		ext.Refs = refs
	}
}

func fromAddressGroupExtList(items []*sgroupsv1.AddressGroupResp_AddressGroupExt) []*sgroupsv1.AddressGroup {
	out := make([]*sgroupsv1.AddressGroup, 0, len(items))
	for _, ag := range items {
		if ag == nil {
			continue
		}
		out = append(out, &sgroupsv1.AddressGroup{
			Metadata: ag.GetMetadata(),
			Spec:     ag.GetSpec(),
		})
	}

	return out
}

func cloneNetwork(nw *sgroupsv1.Network) *sgroupsv1.Network {
	if nw == nil {
		return nil
	}

	return proto.Clone(nw).(*sgroupsv1.Network) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func toNetworkExtList(items []*sgroupsv1.Network) []*sgroupsv1.NetworkResp_NetworkExt {
	out := make([]*sgroupsv1.NetworkResp_NetworkExt, 0, len(items))
	for _, nw := range items {
		if nw == nil {
			continue
		}
		out = append(out, &sgroupsv1.NetworkResp_NetworkExt{
			Metadata: nw.GetMetadata(),
			Spec:     nw.GetSpec(),
		})
	}

	return out
}

// fillNetworkRefs computes refs for each Network by scanning network bindings.
// Must be called while m.mu.RLock is held.
func (m *MockBackend) fillNetworkRefs(extList []*sgroupsv1.NetworkResp_NetworkExt) {
	for _, ext := range extList {
		name := ext.GetMetadata().GetName()
		ns := ext.GetMetadata().GetNamespace()
		var refs []*commonpb.ResourceRef
		for _, nb := range m.networkBindings {
			nw := nb.GetSpec().GetNetwork()
			if nw.GetName() == name && nw.GetNamespace() == ns {
				ag := nb.GetSpec().GetAddressGroup()
				refs = append(refs, &commonpb.ResourceRef{Name: ag.GetName(), Namespace: ag.GetNamespace(), ResType: v1alpha1.KindAddressGroup})
			}
		}
		ext.Refs = refs
	}
}

func cloneHost(h *sgroupsv1.Host) *sgroupsv1.Host {
	if h == nil {
		return nil
	}

	return proto.Clone(h).(*sgroupsv1.Host) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func cloneHostBinding(hb *sgroupsv1.HostBinding) *sgroupsv1.HostBinding {
	if hb == nil {
		return nil
	}

	return proto.Clone(hb).(*sgroupsv1.HostBinding) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func cloneNetworkBinding(nb *sgroupsv1.NetworkBinding) *sgroupsv1.NetworkBinding {
	if nb == nil {
		return nil
	}

	return proto.Clone(nb).(*sgroupsv1.NetworkBinding) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func toHostExtList(items []*sgroupsv1.Host) []*sgroupsv1.HostResp_HostExt {
	out := make([]*sgroupsv1.HostResp_HostExt, 0, len(items))
	for _, h := range items {
		if h == nil {
			continue
		}
		out = append(out, &sgroupsv1.HostResp_HostExt{
			Metadata: h.GetMetadata(),
			Spec:     h.GetSpec(),
		})
	}

	return out
}

// fillHostRefs computes refs for each Host by scanning host bindings.
// Must be called while m.mu.RLock is held.
func (m *MockBackend) fillHostRefs(extList []*sgroupsv1.HostResp_HostExt) {
	for _, ext := range extList {
		name := ext.GetMetadata().GetName()
		ns := ext.GetMetadata().GetNamespace()
		var refs []*commonpb.ResourceRef
		for _, hb := range m.hostBindings {
			h := hb.GetSpec().GetHost()
			if h.GetName() == name && h.GetNamespace() == ns {
				ag := hb.GetSpec().GetAddressGroup()
				refs = append(refs, &commonpb.ResourceRef{Name: ag.GetName(), Namespace: ag.GetNamespace(), ResType: v1alpha1.KindAddressGroup})
			}
		}
		ext.Refs = refs
	}
}

func fromHostExtList(items []*sgroupsv1.HostResp_HostExt) []*sgroupsv1.Host {
	out := make([]*sgroupsv1.Host, 0, len(items))
	for _, h := range items {
		if h == nil {
			continue
		}
		out = append(out, &sgroupsv1.Host{
			Metadata: h.GetMetadata(),
			Spec:     h.GetSpec(),
		})
	}

	return out
}

func filterHosts(selectors []*commonpb.ResSelector, items []*sgroupsv1.Host) []*sgroupsv1.Host {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.Host, 0)
	for _, h := range items {
		if h == nil || h.Metadata == nil {
			continue
		}
		if matchHostSelectors(h, selectors) {
			result = append(result, cloneHost(h))
		}
	}

	return result
}

func matchHostSelectors(h *sgroupsv1.Host, selectors []*commonpb.ResSelector) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchHostSelector(h, sel) {
			return true
		}
	}

	return false
}

func matchHostSelector(h *sgroupsv1.Host, sel *commonpb.ResSelector) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && h.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && h.Metadata.Namespace != fs.Namespace {
			return false
		}
		if len(fs.Refs) > 0 {
			matched := false
			for _, ref := range fs.Refs {
				if ref == nil {
					continue
				}
				if ref.Name != "" && h.Metadata.Name != ref.Name {
					continue
				}
				if ref.Namespace != "" && h.Metadata.Namespace != ref.Namespace {
					continue
				}
				matched = true

				break
			}
			if !matched {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(h.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func filterHostBindings(selectors []*sgroupsv1.HostBindingReq_Selectors, items []*sgroupsv1.HostBinding) []*sgroupsv1.HostBinding {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.HostBinding, 0)
	for _, hb := range items {
		if hb == nil || hb.Metadata == nil {
			continue
		}
		if matchHostBindingSelectors(hb, selectors) {
			result = append(result, cloneHostBinding(hb))
		}
	}

	return result
}

func matchHostBindingSelectors(hb *sgroupsv1.HostBinding, selectors []*sgroupsv1.HostBindingReq_Selectors) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchHostBindingSelector(hb, sel) {
			return true
		}
	}

	return false
}

func matchHostBindingSelector(hb *sgroupsv1.HostBinding, sel *sgroupsv1.HostBindingReq_Selectors) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && hb.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && hb.Metadata.Namespace != fs.Namespace {
			return false
		}
		if fs.AddressGroup != nil {
			ag := hb.GetSpec().GetAddressGroup()
			if ag == nil {
				return false
			}
			if fs.AddressGroup.Name != "" && ag.Name != fs.AddressGroup.Name {
				return false
			}
			if fs.AddressGroup.Namespace != "" && ag.Namespace != fs.AddressGroup.Namespace {
				return false
			}
		}
		if fs.Host != nil {
			h := hb.GetSpec().GetHost()
			if h == nil {
				return false
			}
			if fs.Host.Name != "" && h.Name != fs.Host.Name {
				return false
			}
			if fs.Host.Namespace != "" && h.Namespace != fs.Host.Namespace {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(hb.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func filterNetworkBindings(selectors []*sgroupsv1.NetworkBindingReq_Selectors, items []*sgroupsv1.NetworkBinding) []*sgroupsv1.NetworkBinding {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.NetworkBinding, 0)
	for _, nb := range items {
		if nb == nil || nb.Metadata == nil {
			continue
		}
		if matchNetworkBindingSelectors(nb, selectors) {
			result = append(result, cloneNetworkBinding(nb))
		}
	}

	return result
}

func matchNetworkBindingSelectors(nb *sgroupsv1.NetworkBinding, selectors []*sgroupsv1.NetworkBindingReq_Selectors) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchNetworkBindingSelector(nb, sel) {
			return true
		}
	}

	return false
}

func matchNetworkBindingSelector(nb *sgroupsv1.NetworkBinding, sel *sgroupsv1.NetworkBindingReq_Selectors) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && nb.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && nb.Metadata.Namespace != fs.Namespace {
			return false
		}
		if fs.AddressGroup != nil {
			ag := nb.GetSpec().GetAddressGroup()
			if ag == nil {
				return false
			}
			if fs.AddressGroup.Name != "" && ag.Name != fs.AddressGroup.Name {
				return false
			}
			if fs.AddressGroup.Namespace != "" && ag.Namespace != fs.AddressGroup.Namespace {
				return false
			}
		}
		if fs.Network != nil {
			nw := nb.GetSpec().GetNetwork()
			if nw == nil {
				return false
			}
			if fs.Network.Name != "" && nw.Name != fs.Network.Name {
				return false
			}
			if fs.Network.Namespace != "" && nw.Namespace != fs.Network.Namespace {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(nb.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func filterNamespaces(selectors []*sgroupsv1.NamespaceReq_Selector, items []*sgroupsv1.Namespace) []*sgroupsv1.Namespace {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.Namespace, 0)
	for _, ns := range items {
		if ns == nil || ns.Metadata == nil {
			continue
		}
		if matchNamespaceSelectors(ns, selectors) {
			result = append(result, cloneNamespace(ns))
		}
	}

	return result
}

func matchNamespaceSelectors(ns *sgroupsv1.Namespace, selectors []*sgroupsv1.NamespaceReq_Selector) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchNamespaceSelector(ns, sel) {
			return true
		}
	}

	return false
}

func matchNamespaceSelector(ns *sgroupsv1.Namespace, sel *sgroupsv1.NamespaceReq_Selector) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil && fs.Name != "" {
		if ns.Metadata == nil || ns.Metadata.Name != fs.Name {
			return false
		}
	}
	if len(sel.LabelSelector) > 0 {
		if ns.Metadata == nil || !matchLabels(ns.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func filterAddressGroups(selectors []*commonpb.ResSelector, items []*sgroupsv1.AddressGroup) []*sgroupsv1.AddressGroup {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.AddressGroup, 0)
	for _, ag := range items {
		if ag == nil || ag.Metadata == nil {
			continue
		}
		if matchAddressGroupSelectors(ag, selectors) {
			result = append(result, cloneAddressGroup(ag))
		}
	}

	return result
}

func matchAddressGroupSelectors(ag *sgroupsv1.AddressGroup, selectors []*commonpb.ResSelector) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchAddressGroupSelector(ag, sel) {
			return true
		}
	}

	return false
}

func matchAddressGroupSelector(ag *sgroupsv1.AddressGroup, sel *commonpb.ResSelector) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && ag.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && ag.Metadata.Namespace != fs.Namespace {
			return false
		}
		if len(fs.Refs) > 0 {
			matched := false
			for _, ref := range fs.Refs {
				if ref == nil {
					continue
				}
				if ref.Name != "" && ag.Metadata.Name != ref.Name {
					continue
				}
				if ref.Namespace != "" && ag.Metadata.Namespace != ref.Namespace {
					continue
				}
				matched = true

				break
			}
			if !matched {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(ag.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func filterNetworks(selectors []*commonpb.ResSelector, items []*sgroupsv1.Network) []*sgroupsv1.Network {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.Network, 0)
	for _, nw := range items {
		if nw == nil || nw.Metadata == nil {
			continue
		}
		if matchNetworkSelectors(nw, selectors) {
			result = append(result, cloneNetwork(nw))
		}
	}

	return result
}

func matchNetworkSelectors(nw *sgroupsv1.Network, selectors []*commonpb.ResSelector) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchNetworkSelector(nw, sel) {
			return true
		}
	}

	return false
}

func matchNetworkSelector(nw *sgroupsv1.Network, sel *commonpb.ResSelector) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && nw.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && nw.Metadata.Namespace != fs.Namespace {
			return false
		}
		if len(fs.Refs) > 0 {
			matched := false
			for _, ref := range fs.Refs {
				if ref == nil {
					continue
				}
				if ref.Name != "" && nw.Metadata.Name != ref.Name {
					continue
				}
				if ref.Namespace != "" && nw.Metadata.Namespace != ref.Namespace {
					continue
				}
				matched = true

				break
			}
			if !matched {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(nw.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

func matchLabels(labels map[string]string, selector map[string]string) bool {
	for k, v := range selector {
		if labels == nil || labels[k] != v {
			return false
		}
	}

	return true
}

type namespaceWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*namespaceWatchSub
}

type namespaceWatchSub struct {
	selectors []*sgroupsv1.NamespaceReq_Selector
	ch        chan *sgroupsv1.NamespaceResp_Watch
}

func newNamespaceWatchHub() *namespaceWatchHub {
	return &namespaceWatchHub{subs: make(map[int64]*namespaceWatchSub)}
}

func (h *namespaceWatchHub) subscribe(selectors []*sgroupsv1.NamespaceReq_Selector, buffer int) (chan *sgroupsv1.NamespaceResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.NamespaceResp_Watch, buffer)
	h.subs[id] = &namespaceWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *namespaceWatchHub) publish(event *sgroupsv1.NamespaceResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*namespaceWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterNamespaces(sub.selectors, event.Namespaces)
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.NamespaceResp_Watch{Type: event.Type, Namespaces: filtered}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

type addressGroupWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*addressGroupWatchSub
}

type addressGroupWatchSub struct {
	selectors []*commonpb.ResSelector
	ch        chan *sgroupsv1.AddressGroupResp_Watch
}

func newAddressGroupWatchHub() *addressGroupWatchHub {
	return &addressGroupWatchHub{subs: make(map[int64]*addressGroupWatchSub)}
}

func (h *addressGroupWatchHub) subscribe(selectors []*commonpb.ResSelector, buffer int) (chan *sgroupsv1.AddressGroupResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.AddressGroupResp_Watch, buffer)
	h.subs[id] = &addressGroupWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *addressGroupWatchHub) publish(event *sgroupsv1.AddressGroupResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*addressGroupWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterAddressGroups(sub.selectors, fromAddressGroupExtList(event.AddressGroups))
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.AddressGroupResp_Watch{Type: event.Type, AddressGroups: toAddressGroupExtList(filtered)}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

type networkWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*networkWatchSub
}

type networkWatchSub struct {
	selectors []*commonpb.ResSelector
	ch        chan *sgroupsv1.NetworkResp_Watch
}

func newNetworkWatchHub() *networkWatchHub {
	return &networkWatchHub{subs: make(map[int64]*networkWatchSub)}
}

func (h *networkWatchHub) subscribe(selectors []*commonpb.ResSelector, buffer int) (chan *sgroupsv1.NetworkResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.NetworkResp_Watch, buffer)
	h.subs[id] = &networkWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *networkWatchHub) publish(event *sgroupsv1.NetworkResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*networkWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterNetworks(sub.selectors, fromNetworkExtList(event.Networks))
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.NetworkResp_Watch{Type: event.Type, Networks: toNetworkExtList(filtered)}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

func fromNetworkExtList(items []*sgroupsv1.NetworkResp_NetworkExt) []*sgroupsv1.Network {
	out := make([]*sgroupsv1.Network, 0, len(items))
	for _, nw := range items {
		if nw == nil {
			continue
		}
		out = append(out, &sgroupsv1.Network{
			Metadata: nw.GetMetadata(),
			Spec:     nw.GetSpec(),
		})
	}

	return out
}

type hostWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*hostWatchSub
}

type hostWatchSub struct {
	selectors []*commonpb.ResSelector
	ch        chan *sgroupsv1.HostResp_Watch
}

func newHostWatchHub() *hostWatchHub {
	return &hostWatchHub{subs: make(map[int64]*hostWatchSub)}
}

func (h *hostWatchHub) subscribe(selectors []*commonpb.ResSelector, buffer int) (chan *sgroupsv1.HostResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.HostResp_Watch, buffer)
	h.subs[id] = &hostWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *hostWatchHub) publish(event *sgroupsv1.HostResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*hostWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterHosts(sub.selectors, fromHostExtList(event.Hosts))
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.HostResp_Watch{Type: event.Type, Hosts: toHostExtList(filtered)}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

type hostBindingWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*hostBindingWatchSub
}

type hostBindingWatchSub struct {
	selectors []*sgroupsv1.HostBindingReq_Selectors
	ch        chan *sgroupsv1.HostBindingResp_Watch
}

func newHostBindingWatchHub() *hostBindingWatchHub {
	return &hostBindingWatchHub{subs: make(map[int64]*hostBindingWatchSub)}
}

func (h *hostBindingWatchHub) subscribe(selectors []*sgroupsv1.HostBindingReq_Selectors, buffer int) (chan *sgroupsv1.HostBindingResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.HostBindingResp_Watch, buffer)
	h.subs[id] = &hostBindingWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *hostBindingWatchHub) publish(event *sgroupsv1.HostBindingResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*hostBindingWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterHostBindings(sub.selectors, event.HostBindings)
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.HostBindingResp_Watch{Type: event.Type, HostBindings: filtered}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

type networkBindingWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*networkBindingWatchSub
}

type networkBindingWatchSub struct {
	selectors []*sgroupsv1.NetworkBindingReq_Selectors
	ch        chan *sgroupsv1.NetworkBindingResp_Watch
}

func newNetworkBindingWatchHub() *networkBindingWatchHub {
	return &networkBindingWatchHub{subs: make(map[int64]*networkBindingWatchSub)}
}

func (h *networkBindingWatchHub) subscribe(selectors []*sgroupsv1.NetworkBindingReq_Selectors, buffer int) (chan *sgroupsv1.NetworkBindingResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.NetworkBindingResp_Watch, buffer)
	h.subs[id] = &networkBindingWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *networkBindingWatchHub) publish(event *sgroupsv1.NetworkBindingResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*networkBindingWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterNetworkBindings(sub.selectors, event.NetworkBindings)
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.NetworkBindingResp_Watch{Type: event.Type, NetworkBindings: filtered}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

// ---- Service CRUD ----

func (m *MockBackend) UpsertServices(ctx context.Context, req *sgroupsv1.ServiceReq_Upsert) (*sgroupsv1.ServiceResp_Upsert, error) {
	if req == nil || len(req.Services) == 0 {
		return nil, errors.New("services are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.Service, 0, len(req.Services))
	modified := make([]*sgroupsv1.Service, 0, len(req.Services))
	resp := &sgroupsv1.ServiceResp_Upsert{Services: make([]*sgroupsv1.Service, 0, len(req.Services))}
	for _, svc := range req.Services {
		if svc == nil || svc.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("service metadata is required")
		}
		stored, isNew := m.upsertServiceLocked(svc)
		resp.Services = append(resp.Services, cloneService(stored))
		if isNew {
			added = append(added, cloneService(stored))
		} else {
			modified = append(modified, cloneService(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.svcHub.publish(&sgroupsv1.ServiceResp_Watch{Type: commonpb.WatchEventType_ADDED, Services: toServiceExtList(added)})
	}
	if len(modified) > 0 {
		m.svcHub.publish(&sgroupsv1.ServiceResp_Watch{Type: commonpb.WatchEventType_MODIFIED, Services: toServiceExtList(modified)})
	}

	return resp, nil
}

func (m *MockBackend) DeleteServices(ctx context.Context, req *sgroupsv1.ServiceReq_Delete) error {
	if req == nil || len(req.Services) == 0 {
		return errors.New("services are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.Service, 0, len(req.Services))
	for _, svc := range req.Services {
		if svc == nil || svc.Metadata == nil {
			m.mu.Unlock()

			return errors.New("service delete metadata is required")
		}
		uid := svc.Metadata.Uid
		name := svc.Metadata.Name
		ns := svc.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("service delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.services[uid]; ok {
				delete(m.services, uid)
				deleted = append(deleted, cloneService(stored))
			}

			continue
		}
		for id, stored := range m.services {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.services, id)
				deleted = append(deleted, cloneService(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.svcHub.publish(&sgroupsv1.ServiceResp_Watch{Type: commonpb.WatchEventType_DELETED, Services: toServiceExtList(deleted)})
	}

	return nil
}

func (m *MockBackend) ListServices(ctx context.Context, req *sgroupsv1.ServiceReq_List) (*sgroupsv1.ServiceResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.Service, 0, len(m.services))
	for _, svc := range m.services {
		items = append(items, svc)
	}

	filtered := filterServices(req.Selectors, items)
	extList := toServiceExtList(filtered)
	m.fillServiceRefs(extList)
	m.mu.RUnlock()

	return &sgroupsv1.ServiceResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		Services:        extList,
	}, nil
}

func (m *MockBackend) WatchServices(ctx context.Context, req *sgroupsv1.ServiceReq_Watch) (backend.WatchStream[*sgroupsv1.ServiceResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.ServiceResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.svcHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.Service, 0, len(m.services))
	for _, svc := range m.services {
		items = append(items, svc)
	}
	m.mu.RUnlock()

	snapshot := filterServices(req.Selectors, items)
	ch <- &sgroupsv1.ServiceResp_Watch{Type: commonpb.WatchEventType_ADDED, Services: toServiceExtList(snapshot)}

	return backend.WatchStream[*sgroupsv1.ServiceResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) upsertServiceLocked(svc *sgroupsv1.Service) (*sgroupsv1.Service, bool) {
	clone := cloneService(svc)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.services[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.services[uid] = clone

	return clone, !ok
}

func cloneService(svc *sgroupsv1.Service) *sgroupsv1.Service {
	if svc == nil {
		return nil
	}

	return proto.Clone(svc).(*sgroupsv1.Service) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func toServiceExtList(items []*sgroupsv1.Service) []*sgroupsv1.ServiceResp_ServiceExt {
	out := make([]*sgroupsv1.ServiceResp_ServiceExt, 0, len(items))
	for _, svc := range items {
		if svc == nil {
			continue
		}
		out = append(out, &sgroupsv1.ServiceResp_ServiceExt{
			Metadata: svc.GetMetadata(),
			Spec:     svc.GetSpec(),
		})
	}

	return out
}

// fillServiceRefs computes refs for each Service by scanning service bindings.
// Must be called while m.mu.RLock is held.
func (m *MockBackend) fillServiceRefs(extList []*sgroupsv1.ServiceResp_ServiceExt) {
	for _, ext := range extList {
		name := ext.GetMetadata().GetName()
		ns := ext.GetMetadata().GetNamespace()
		var refs []*commonpb.ResourceRef
		for _, sb := range m.serviceBindings {
			svc := sb.GetSpec().GetService()
			if svc.GetName() == name && svc.GetNamespace() == ns {
				ag := sb.GetSpec().GetAddressGroup()
				refs = append(refs, &commonpb.ResourceRef{Name: ag.GetName(), Namespace: ag.GetNamespace(), ResType: v1alpha1.KindAddressGroup})
			}
		}
		ext.Refs = refs
	}
}

func fromServiceExtList(items []*sgroupsv1.ServiceResp_ServiceExt) []*sgroupsv1.Service {
	out := make([]*sgroupsv1.Service, 0, len(items))
	for _, svc := range items {
		if svc == nil {
			continue
		}
		out = append(out, &sgroupsv1.Service{
			Metadata: svc.GetMetadata(),
			Spec:     svc.GetSpec(),
		})
	}

	return out
}

func filterServices(selectors []*commonpb.ResSelector, items []*sgroupsv1.Service) []*sgroupsv1.Service {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.Service, 0)
	for _, svc := range items {
		if svc == nil || svc.Metadata == nil {
			continue
		}
		if matchServiceSelectors(svc, selectors) {
			result = append(result, cloneService(svc))
		}
	}

	return result
}

func matchServiceSelectors(svc *sgroupsv1.Service, selectors []*commonpb.ResSelector) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchServiceSelector(svc, sel) {
			return true
		}
	}

	return false
}

func matchServiceSelector(svc *sgroupsv1.Service, sel *commonpb.ResSelector) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && svc.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && svc.Metadata.Namespace != fs.Namespace {
			return false
		}
		if len(fs.Refs) > 0 {
			matched := false
			for _, ref := range fs.Refs {
				if ref == nil {
					continue
				}
				if ref.Name != "" && svc.Metadata.Name != ref.Name {
					continue
				}
				if ref.Namespace != "" && svc.Metadata.Namespace != ref.Namespace {
					continue
				}
				matched = true

				break
			}
			if !matched {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(svc.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

// ---- Service Watch Hub ----

type serviceWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*serviceWatchSub
}

type serviceWatchSub struct {
	selectors []*commonpb.ResSelector
	ch        chan *sgroupsv1.ServiceResp_Watch
}

func newServiceWatchHub() *serviceWatchHub {
	return &serviceWatchHub{subs: make(map[int64]*serviceWatchSub)}
}

func (h *serviceWatchHub) subscribe(selectors []*commonpb.ResSelector, buffer int) (chan *sgroupsv1.ServiceResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.ServiceResp_Watch, buffer)
	h.subs[id] = &serviceWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *serviceWatchHub) publish(event *sgroupsv1.ServiceResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*serviceWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterServices(sub.selectors, fromServiceExtList(event.Services))
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.ServiceResp_Watch{Type: event.Type, Services: toServiceExtList(filtered)}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

// ---- ServiceBinding CRUD ----

func (m *MockBackend) UpsertServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Upsert) (*sgroupsv1.ServiceBindingResp_Upsert, error) {
	if req == nil || len(req.ServiceBindings) == 0 {
		return nil, errors.New("service bindings are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.ServiceBinding, 0, len(req.ServiceBindings))
	modified := make([]*sgroupsv1.ServiceBinding, 0, len(req.ServiceBindings))
	resp := &sgroupsv1.ServiceBindingResp_Upsert{ServiceBindings: make([]*sgroupsv1.ServiceBinding, 0, len(req.ServiceBindings))}
	for _, sb := range req.ServiceBindings {
		if sb == nil || sb.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("service binding metadata is required")
		}
		stored, isNew := m.upsertServiceBindingLocked(sb)
		resp.ServiceBindings = append(resp.ServiceBindings, cloneServiceBinding(stored))
		if isNew {
			added = append(added, cloneServiceBinding(stored))
		} else {
			modified = append(modified, cloneServiceBinding(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.sbHub.publish(&sgroupsv1.ServiceBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, ServiceBindings: added})
	}
	if len(modified) > 0 {
		m.sbHub.publish(&sgroupsv1.ServiceBindingResp_Watch{Type: commonpb.WatchEventType_MODIFIED, ServiceBindings: modified})
	}

	return resp, nil
}

func (m *MockBackend) DeleteServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Delete) error {
	if req == nil || len(req.ServiceBindings) == 0 {
		return errors.New("service bindings are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.ServiceBinding, 0, len(req.ServiceBindings))
	for _, sb := range req.ServiceBindings {
		if sb == nil || sb.Metadata == nil {
			m.mu.Unlock()

			return errors.New("service binding delete metadata is required")
		}
		uid := sb.Metadata.Uid
		name := sb.Metadata.Name
		ns := sb.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("service binding delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.serviceBindings[uid]; ok {
				delete(m.serviceBindings, uid)
				deleted = append(deleted, cloneServiceBinding(stored))
			}

			continue
		}
		for id, stored := range m.serviceBindings {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.serviceBindings, id)
				deleted = append(deleted, cloneServiceBinding(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.sbHub.publish(&sgroupsv1.ServiceBindingResp_Watch{Type: commonpb.WatchEventType_DELETED, ServiceBindings: deleted})
	}

	return nil
}

func (m *MockBackend) ListServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_List) (*sgroupsv1.ServiceBindingResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.ServiceBinding, 0, len(m.serviceBindings))
	for _, sb := range m.serviceBindings {
		items = append(items, sb)
	}
	m.mu.RUnlock()

	filtered := filterServiceBindings(req.Selectors, items)

	return &sgroupsv1.ServiceBindingResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		ServiceBindings: filtered,
	}, nil
}

func (m *MockBackend) WatchServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Watch) (backend.WatchStream[*sgroupsv1.ServiceBindingResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.ServiceBindingResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.sbHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.ServiceBinding, 0, len(m.serviceBindings))
	for _, sb := range m.serviceBindings {
		items = append(items, sb)
	}
	m.mu.RUnlock()

	snapshot := filterServiceBindings(req.Selectors, items)
	ch <- &sgroupsv1.ServiceBindingResp_Watch{Type: commonpb.WatchEventType_ADDED, ServiceBindings: snapshot}

	return backend.WatchStream[*sgroupsv1.ServiceBindingResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) upsertServiceBindingLocked(sb *sgroupsv1.ServiceBinding) (*sgroupsv1.ServiceBinding, bool) {
	clone := cloneServiceBinding(sb)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.serviceBindings[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.serviceBindings[uid] = clone

	return clone, !ok
}

func cloneServiceBinding(sb *sgroupsv1.ServiceBinding) *sgroupsv1.ServiceBinding {
	if sb == nil {
		return nil
	}

	return proto.Clone(sb).(*sgroupsv1.ServiceBinding) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func filterServiceBindings(selectors []*sgroupsv1.ServiceBindingReq_Selectors, items []*sgroupsv1.ServiceBinding) []*sgroupsv1.ServiceBinding {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.ServiceBinding, 0)
	for _, sb := range items {
		if sb == nil || sb.Metadata == nil {
			continue
		}
		if matchServiceBindingSelectors(sb, selectors) {
			result = append(result, cloneServiceBinding(sb))
		}
	}

	return result
}

func matchServiceBindingSelectors(sb *sgroupsv1.ServiceBinding, selectors []*sgroupsv1.ServiceBindingReq_Selectors) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchServiceBindingSelector(sb, sel) {
			return true
		}
	}

	return false
}

func matchServiceBindingSelector(sb *sgroupsv1.ServiceBinding, sel *sgroupsv1.ServiceBindingReq_Selectors) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && sb.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && sb.Metadata.Namespace != fs.Namespace {
			return false
		}
		if fs.AddressGroup != nil {
			ag := sb.GetSpec().GetAddressGroup()
			if ag == nil {
				return false
			}
			if fs.AddressGroup.Name != "" && ag.Name != fs.AddressGroup.Name {
				return false
			}
			if fs.AddressGroup.Namespace != "" && ag.Namespace != fs.AddressGroup.Namespace {
				return false
			}
		}
		if fs.Service != nil {
			svc := sb.GetSpec().GetService()
			if svc == nil {
				return false
			}
			if fs.Service.Name != "" && svc.Name != fs.Service.Name {
				return false
			}
			if fs.Service.Namespace != "" && svc.Namespace != fs.Service.Namespace {
				return false
			}
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(sb.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

// ---- ServiceBinding Watch Hub ----

type serviceBindingWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*serviceBindingWatchSub
}

type serviceBindingWatchSub struct {
	selectors []*sgroupsv1.ServiceBindingReq_Selectors
	ch        chan *sgroupsv1.ServiceBindingResp_Watch
}

func newServiceBindingWatchHub() *serviceBindingWatchHub {
	return &serviceBindingWatchHub{subs: make(map[int64]*serviceBindingWatchSub)}
}

func (h *serviceBindingWatchHub) subscribe(selectors []*sgroupsv1.ServiceBindingReq_Selectors, buffer int) (chan *sgroupsv1.ServiceBindingResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.ServiceBindingResp_Watch, buffer)
	h.subs[id] = &serviceBindingWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *serviceBindingWatchHub) publish(event *sgroupsv1.ServiceBindingResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*serviceBindingWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterServiceBindings(sub.selectors, event.ServiceBindings)
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.ServiceBindingResp_Watch{Type: event.Type, ServiceBindings: filtered}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}

// ---- Rule CRUD ----

func (m *MockBackend) UpsertRules(ctx context.Context, req *sgroupsv1.RuleReq_Upsert) (*sgroupsv1.RuleResp_Upsert, error) {
	if req == nil || len(req.Rules) == 0 {
		return nil, errors.New("rules are required")
	}
	m.mu.Lock()
	added := make([]*sgroupsv1.Rule, 0, len(req.Rules))
	modified := make([]*sgroupsv1.Rule, 0, len(req.Rules))
	resp := &sgroupsv1.RuleResp_Upsert{Rules: make([]*sgroupsv1.Rule, 0, len(req.Rules))}
	for _, r := range req.Rules {
		if r == nil || r.Metadata == nil {
			m.mu.Unlock()

			return nil, errors.New("rule metadata is required")
		}
		stored, isNew := m.upsertRuleLocked(r)
		resp.Rules = append(resp.Rules, cloneRule(stored))
		if isNew {
			added = append(added, cloneRule(stored))
		} else {
			modified = append(modified, cloneRule(stored))
		}
	}
	m.mu.Unlock()

	if len(added) > 0 {
		m.ruleHub.publish(&sgroupsv1.RuleResp_Watch{Type: commonpb.WatchEventType_ADDED, Rules: added})
	}
	if len(modified) > 0 {
		m.ruleHub.publish(&sgroupsv1.RuleResp_Watch{Type: commonpb.WatchEventType_MODIFIED, Rules: modified})
	}

	return resp, nil
}

func (m *MockBackend) DeleteRules(ctx context.Context, req *sgroupsv1.RuleReq_Delete) error {
	if req == nil || len(req.Rules) == 0 {
		return errors.New("rules are required")
	}
	m.mu.Lock()
	deleted := make([]*sgroupsv1.Rule, 0, len(req.Rules))
	for _, r := range req.Rules {
		if r == nil || r.Metadata == nil {
			m.mu.Unlock()

			return errors.New("rule delete metadata is required")
		}
		uid := r.Metadata.Uid
		name := r.Metadata.Name
		ns := r.Metadata.Namespace
		if uid == "" && (name == "" || ns == "") {
			m.mu.Unlock()

			return errors.New("rule delete requires uid or name+namespace")
		}
		if uid != "" {
			if stored, ok := m.rules[uid]; ok {
				delete(m.rules, uid)
				deleted = append(deleted, cloneRule(stored))
			}

			continue
		}
		for id, stored := range m.rules {
			if stored.Metadata != nil && stored.Metadata.Name == name && stored.Metadata.Namespace == ns {
				delete(m.rules, id)
				deleted = append(deleted, cloneRule(stored))

				break
			}
		}
	}
	m.mu.Unlock()

	if len(deleted) > 0 {
		m.ruleHub.publish(&sgroupsv1.RuleResp_Watch{Type: commonpb.WatchEventType_DELETED, Rules: deleted})
	}

	return nil
}

func (m *MockBackend) ListRules(ctx context.Context, req *sgroupsv1.RuleReq_List) (*sgroupsv1.RuleResp_List, error) {
	if req == nil || len(req.Selectors) == 0 {
		return nil, errors.New("selectors are required")
	}
	m.mu.RLock()
	items := make([]*sgroupsv1.Rule, 0, len(m.rules))
	for _, r := range m.rules {
		items = append(items, r)
	}
	m.mu.RUnlock()

	filtered := filterRules(req.Selectors, items)

	return &sgroupsv1.RuleResp_List{
		ResourceVersion: strconv.FormatInt(m.version.Load(), 10),
		Rules:           filtered,
	}, nil
}

func (m *MockBackend) WatchRules(ctx context.Context, req *sgroupsv1.RuleReq_Watch) (backend.WatchStream[*sgroupsv1.RuleResp_Watch], error) {
	if req == nil || len(req.Selectors) == 0 {
		return backend.WatchStream[*sgroupsv1.RuleResp_Watch]{}, errors.New("selectors are required")
	}
	ch, cancel := m.ruleHub.subscribe(req.Selectors, 16)

	m.mu.RLock()
	items := make([]*sgroupsv1.Rule, 0, len(m.rules))
	for _, r := range m.rules {
		items = append(items, r)
	}
	m.mu.RUnlock()

	snapshot := filterRules(req.Selectors, items)
	ch <- &sgroupsv1.RuleResp_Watch{Type: commonpb.WatchEventType_ADDED, Rules: snapshot}

	return backend.WatchStream[*sgroupsv1.RuleResp_Watch]{
		C:     ch,
		Close: cancel,
	}, nil
}

func (m *MockBackend) upsertRuleLocked(r *sgroupsv1.Rule) (*sgroupsv1.Rule, bool) {
	clone := cloneRule(r)
	uid := clone.Metadata.Uid
	if uid == "" {
		uid = uuid.NewString()
		clone.Metadata.Uid = uid
	}
	existing, ok := m.rules[uid]
	if ok && existing != nil && existing.Metadata != nil {
		clone.Metadata.CreationTimestamp = existing.Metadata.CreationTimestamp
	} else if clone.Metadata.CreationTimestamp == nil {
		clone.Metadata.CreationTimestamp = timestamppb.Now()
	}
	clone.Metadata.ResourceVersion = strconv.FormatInt(m.version.Add(1), 10)
	m.rules[uid] = clone

	return clone, !ok
}

func cloneRule(r *sgroupsv1.Rule) *sgroupsv1.Rule {
	if r == nil {
		return nil
	}

	return proto.Clone(r).(*sgroupsv1.Rule) //nolint:forcetypeassert,errcheck // proto.Clone preserves concrete type
}

func filterRules(selectors []*sgroupsv1.RuleReq_Selectors, items []*sgroupsv1.Rule) []*sgroupsv1.Rule {
	if len(selectors) == 0 {
		return nil
	}
	result := make([]*sgroupsv1.Rule, 0)
	for _, r := range items {
		if r == nil || r.Metadata == nil {
			continue
		}
		if matchRuleSelectors(r, selectors) {
			result = append(result, cloneRule(r))
		}
	}

	return result
}

func matchRuleSelectors(r *sgroupsv1.Rule, selectors []*sgroupsv1.RuleReq_Selectors) bool {
	for _, sel := range selectors {
		if sel == nil {
			continue
		}
		if matchRuleSelector(r, sel) {
			return true
		}
	}

	return false
}

func matchRuleSelector(r *sgroupsv1.Rule, sel *sgroupsv1.RuleReq_Selectors) bool {
	if sel == nil {
		return false
	}
	if fs := sel.FieldSelector; fs != nil {
		if fs.Name != "" && r.Metadata.Name != fs.Name {
			return false
		}
		if fs.Namespace != "" && r.Metadata.Namespace != fs.Namespace {
			return false
		}
	}
	if len(sel.LabelSelector) > 0 {
		if !matchLabels(r.Metadata.Labels, sel.LabelSelector) {
			return false
		}
	}

	return true
}

// ---- Rule Watch Hub ----

type ruleWatchHub struct {
	mu     sync.Mutex
	nextID int64
	subs   map[int64]*ruleWatchSub
}

type ruleWatchSub struct {
	selectors []*sgroupsv1.RuleReq_Selectors
	ch        chan *sgroupsv1.RuleResp_Watch
}

func newRuleWatchHub() *ruleWatchHub {
	return &ruleWatchHub{subs: make(map[int64]*ruleWatchSub)}
}

func (h *ruleWatchHub) subscribe(selectors []*sgroupsv1.RuleReq_Selectors, buffer int) (chan *sgroupsv1.RuleResp_Watch, func()) {
	if buffer <= 0 {
		buffer = 1
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	ch := make(chan *sgroupsv1.RuleResp_Watch, buffer)
	h.subs[id] = &ruleWatchSub{selectors: selectors, ch: ch}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}
}

func (h *ruleWatchHub) publish(event *sgroupsv1.RuleResp_Watch) {
	if event == nil {
		return
	}
	h.mu.Lock()
	subs := make([]*ruleWatchSub, 0, len(h.subs))
	for _, sub := range h.subs {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	for _, sub := range subs {
		filtered := filterRules(sub.selectors, event.Rules)
		if len(filtered) == 0 {
			continue
		}
		resp := &sgroupsv1.RuleResp_Watch{Type: event.Type, Rules: filtered}
		select {
		case sub.ch <- resp:
		default:
		}
	}
}
