package backend

import (
	"context"

	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
)

type WatchStream[T any] struct {
	C     <-chan T
	Close func()
}

type NamespaceBackend interface {
	UpsertNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Upsert) (*sgroupsv1.NamespaceResp_Upsert, error)
	DeleteNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Delete) error
	ListNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_List) (*sgroupsv1.NamespaceResp_List, error)
	WatchNamespaces(ctx context.Context, req *sgroupsv1.NamespaceReq_Watch) (WatchStream[*sgroupsv1.NamespaceResp_Watch], error)
}

type AddressGroupBackend interface {
	UpsertAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Upsert) (*sgroupsv1.AddressGroupResp_Upsert, error)
	DeleteAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Delete) error
	ListAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_List) (*sgroupsv1.AddressGroupResp_List, error)
	WatchAddressGroups(ctx context.Context, req *sgroupsv1.AddressGroupReq_Watch) (WatchStream[*sgroupsv1.AddressGroupResp_Watch], error)
}

type NetworkBackend interface {
	UpsertNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Upsert) (*sgroupsv1.NetworkResp_Upsert, error)
	DeleteNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Delete) error
	ListNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_List) (*sgroupsv1.NetworkResp_List, error)
	WatchNetworks(ctx context.Context, req *sgroupsv1.NetworkReq_Watch) (WatchStream[*sgroupsv1.NetworkResp_Watch], error)
}

type HostBackend interface {
	UpsertHosts(ctx context.Context, req *sgroupsv1.HostReq_Upsert) (*sgroupsv1.HostResp_Upsert, error)
	DeleteHosts(ctx context.Context, req *sgroupsv1.HostReq_Delete) error
	ListHosts(ctx context.Context, req *sgroupsv1.HostReq_List) (*sgroupsv1.HostResp_List, error)
	WatchHosts(ctx context.Context, req *sgroupsv1.HostReq_Watch) (WatchStream[*sgroupsv1.HostResp_Watch], error)
	ListSocketStatistics(
		ctx context.Context, req *sgroupsv1.HostReq_SocketStatistics_List,
	) (*sgroupsv1.HostResp_SocketStatistics_List, error)
	WatchSocketStatistics(
		ctx context.Context, req *sgroupsv1.HostReq_SocketStatistics_Watch,
	) (WatchStream[*sgroupsv1.HostResp_SocketStatistics_Watch], error)
	ListNft(
		ctx context.Context, req *sgroupsv1.HostReq_Nft_List,
	) (*sgroupsv1.HostResp_Nft_List, error)
	WatchNft(
		ctx context.Context, req *sgroupsv1.HostReq_Nft_Watch,
	) (WatchStream[*sgroupsv1.HostResp_Nft_Watch], error)
}

type HostBindingBackend interface {
	UpsertHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Upsert) (*sgroupsv1.HostBindingResp_Upsert, error)
	DeleteHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Delete) error
	ListHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_List) (*sgroupsv1.HostBindingResp_List, error)
	WatchHostBindings(ctx context.Context, req *sgroupsv1.HostBindingReq_Watch) (WatchStream[*sgroupsv1.HostBindingResp_Watch], error)
}

type NetworkBindingBackend interface {
	UpsertNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Upsert) (*sgroupsv1.NetworkBindingResp_Upsert, error)
	DeleteNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Delete) error
	ListNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_List) (*sgroupsv1.NetworkBindingResp_List, error)
	WatchNetworkBindings(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Watch) (WatchStream[*sgroupsv1.NetworkBindingResp_Watch], error)
}

type ServiceBackend interface {
	UpsertServices(ctx context.Context, req *sgroupsv1.ServiceReq_Upsert) (*sgroupsv1.ServiceResp_Upsert, error)
	DeleteServices(ctx context.Context, req *sgroupsv1.ServiceReq_Delete) error
	ListServices(ctx context.Context, req *sgroupsv1.ServiceReq_List) (*sgroupsv1.ServiceResp_List, error)
	WatchServices(ctx context.Context, req *sgroupsv1.ServiceReq_Watch) (WatchStream[*sgroupsv1.ServiceResp_Watch], error)
}

type ServiceBindingBackend interface {
	UpsertServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Upsert) (*sgroupsv1.ServiceBindingResp_Upsert, error)
	DeleteServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Delete) error
	ListServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_List) (*sgroupsv1.ServiceBindingResp_List, error)
	WatchServiceBindings(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Watch) (WatchStream[*sgroupsv1.ServiceBindingResp_Watch], error)
}

type RuleBackend interface {
	UpsertRules(ctx context.Context, req *sgroupsv1.RuleReq_Upsert) (*sgroupsv1.RuleResp_Upsert, error)
	DeleteRules(ctx context.Context, req *sgroupsv1.RuleReq_Delete) error
	ListRules(ctx context.Context, req *sgroupsv1.RuleReq_List) (*sgroupsv1.RuleResp_List, error)
	WatchRules(ctx context.Context, req *sgroupsv1.RuleReq_Watch) (WatchStream[*sgroupsv1.RuleResp_Watch], error)
}

type Backend struct {
	Namespaces      NamespaceBackend
	AddressGroups   AddressGroupBackend
	Networks        NetworkBackend
	Hosts           HostBackend
	HostBindings    HostBindingBackend
	NetworkBindings NetworkBindingBackend
	Services        ServiceBackend
	ServiceBindings ServiceBindingBackend
	Rules           RuleBackend
}
