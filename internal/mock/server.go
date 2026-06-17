package mock

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"sgroups.io/sgroups-k8s-api/internal/backend"

	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
)

// MockNamespaceServer implements the SGroupsNamespaceAPI for tests and local runs.
type MockNamespaceServer struct {
	sgroupsv1.UnimplementedSGroupsNamespaceAPIServer
	backend backend.NamespaceBackend
}

// MockAddressGroupServer implements the SGroupsAddressGroupsAPI for tests and local runs.
type MockAddressGroupServer struct {
	sgroupsv1.UnimplementedSGroupsAddressGroupsAPIServer
	backend backend.AddressGroupBackend
}

// MockNetworkServer implements the SGroupsNetworksAPI for tests and local runs.
type MockNetworkServer struct {
	sgroupsv1.UnimplementedSGroupsNetworksAPIServer
	backend backend.NetworkBackend
}

// MockHostServer implements the SGroupsHostsAPI for tests and local runs.
type MockHostServer struct {
	sgroupsv1.UnimplementedSGroupsHostsAPIServer
	backend backend.HostBackend
}

// MockHostBindingServer implements the SGroupsHostBindingAPI for tests and local runs.
type MockHostBindingServer struct {
	sgroupsv1.UnimplementedSGroupsHostBindingAPIServer
	backend backend.HostBindingBackend
}

// MockNetworkBindingServer implements the SGroupsNetworkBindingAPI for tests and local runs.
type MockNetworkBindingServer struct {
	sgroupsv1.UnimplementedSGroupsNetworkBindingAPIServer
	backend backend.NetworkBindingBackend
}

// MockServiceServer implements the SGroupsServicesAPI for tests and local runs.
type MockServiceServer struct {
	sgroupsv1.UnimplementedSGroupsServicesAPIServer
	backend backend.ServiceBackend
}

// MockServiceBindingServer implements the SGroupsServiceBindingAPI for tests and local runs.
type MockServiceBindingServer struct {
	sgroupsv1.UnimplementedSGroupsServiceBindingAPIServer
	backend backend.ServiceBindingBackend
}

// Run starts the mock gRPC server.
func Run(ctx context.Context, addr string, b backend.Backend) error {
	lis, err := net.Listen("tcp", addr) //nolint:noctx // mock server for local dev, no need for ListenConfig
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	RegisterServices(grpcServer, b)
	reflection.Register(grpcServer)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
}

// RegisterServices registers gRPC services backed by the provided backend.
func RegisterServices(grpcServer *grpc.Server, b backend.Backend) {
	sgroupsv1.RegisterSGroupsNamespaceAPIServer(grpcServer, &MockNamespaceServer{backend: b.Namespaces})
	sgroupsv1.RegisterSGroupsAddressGroupsAPIServer(grpcServer, &MockAddressGroupServer{backend: b.AddressGroups})
	sgroupsv1.RegisterSGroupsNetworksAPIServer(grpcServer, &MockNetworkServer{backend: b.Networks})
	sgroupsv1.RegisterSGroupsHostsAPIServer(grpcServer, &MockHostServer{backend: b.Hosts})
	sgroupsv1.RegisterSGroupsHostBindingAPIServer(grpcServer, &MockHostBindingServer{backend: b.HostBindings})
	sgroupsv1.RegisterSGroupsNetworkBindingAPIServer(grpcServer, &MockNetworkBindingServer{backend: b.NetworkBindings})
	sgroupsv1.RegisterSGroupsServicesAPIServer(grpcServer, &MockServiceServer{backend: b.Services})
	sgroupsv1.RegisterSGroupsServiceBindingAPIServer(grpcServer, &MockServiceBindingServer{backend: b.ServiceBindings})
}

func (s *MockNamespaceServer) Upsert(ctx context.Context, req *sgroupsv1.NamespaceReq_Upsert) (*sgroupsv1.NamespaceResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "namespace backend is not configured")
	}
	if req == nil || len(req.Namespaces) == 0 {
		return nil, status.Error(codes.InvalidArgument, "namespaces are required")
	}

	return s.backend.UpsertNamespaces(ctx, req)
}

func (s *MockNamespaceServer) Delete(ctx context.Context, req *sgroupsv1.NamespaceReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "namespace backend is not configured")
	}
	if req == nil || len(req.Namespaces) == 0 {
		return nil, status.Error(codes.InvalidArgument, "namespaces are required")
	}
	if err := s.backend.DeleteNamespaces(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockNamespaceServer) List(ctx context.Context, req *sgroupsv1.NamespaceReq_List) (*sgroupsv1.NamespaceResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "namespace backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListNamespaces(ctx, req)
}

func (s *MockNamespaceServer) Watch(req *sgroupsv1.NamespaceReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.NamespaceResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "namespace backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchNamespaces(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockAddressGroupServer) Upsert(ctx context.Context, req *sgroupsv1.AddressGroupReq_Upsert) (*sgroupsv1.AddressGroupResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "address group backend is not configured")
	}
	if req == nil || len(req.AddressGroups) == 0 {
		return nil, status.Error(codes.InvalidArgument, "address groups are required")
	}

	return s.backend.UpsertAddressGroups(ctx, req)
}

func (s *MockAddressGroupServer) Delete(ctx context.Context, req *sgroupsv1.AddressGroupReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "address group backend is not configured")
	}
	if req == nil || len(req.AddressGroups) == 0 {
		return nil, status.Error(codes.InvalidArgument, "address groups are required")
	}
	if err := s.backend.DeleteAddressGroups(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockAddressGroupServer) List(ctx context.Context, req *sgroupsv1.AddressGroupReq_List) (*sgroupsv1.AddressGroupResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "address group backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListAddressGroups(ctx, req)
}

func (s *MockAddressGroupServer) Watch(req *sgroupsv1.AddressGroupReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.AddressGroupResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "address group backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchAddressGroups(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockNetworkServer) Upsert(ctx context.Context, req *sgroupsv1.NetworkReq_Upsert) (*sgroupsv1.NetworkResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network backend is not configured")
	}
	if req == nil || len(req.Networks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "networks are required")
	}

	return s.backend.UpsertNetworks(ctx, req)
}

func (s *MockNetworkServer) Delete(ctx context.Context, req *sgroupsv1.NetworkReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network backend is not configured")
	}
	if req == nil || len(req.Networks) == 0 {
		return nil, status.Error(codes.InvalidArgument, "networks are required")
	}
	if err := s.backend.DeleteNetworks(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockNetworkServer) List(ctx context.Context, req *sgroupsv1.NetworkReq_List) (*sgroupsv1.NetworkResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListNetworks(ctx, req)
}

func (s *MockNetworkServer) Watch(req *sgroupsv1.NetworkReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.NetworkResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "network backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchNetworks(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockHostServer) Upsert(ctx context.Context, req *sgroupsv1.HostReq_Upsert) (*sgroupsv1.HostResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Hosts) == 0 {
		return nil, status.Error(codes.InvalidArgument, "hosts are required")
	}

	return s.backend.UpsertHosts(ctx, req)
}

func (s *MockHostServer) Delete(ctx context.Context, req *sgroupsv1.HostReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Hosts) == 0 {
		return nil, status.Error(codes.InvalidArgument, "hosts are required")
	}
	if err := s.backend.DeleteHosts(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockHostServer) List(ctx context.Context, req *sgroupsv1.HostReq_List) (*sgroupsv1.HostResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListHosts(ctx, req)
}

func (s *MockHostServer) Watch(req *sgroupsv1.HostReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.HostResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchHosts(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockHostServer) ListSocketStatistics(ctx context.Context, req *sgroupsv1.HostReq_SocketStatistics_List) (*sgroupsv1.HostResp_SocketStatistics_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListSocketStatistics(ctx, req)
}

func (s *MockHostServer) WatchSocketStatistics(req *sgroupsv1.HostReq_SocketStatistics_Watch, stream grpc.ServerStreamingServer[sgroupsv1.HostResp_SocketStatistics_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchSocketStatistics(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockHostServer) ListNft(ctx context.Context, req *sgroupsv1.HostReq_Nft_List) (*sgroupsv1.HostResp_Nft_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListNft(ctx, req)
}

func (s *MockHostServer) WatchNft(req *sgroupsv1.HostReq_Nft_Watch, stream grpc.ServerStreamingServer[sgroupsv1.HostResp_Nft_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "host backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchNft(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockHostBindingServer) Upsert(ctx context.Context, req *sgroupsv1.HostBindingReq_Upsert) (*sgroupsv1.HostBindingResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host binding backend is not configured")
	}
	if req == nil || len(req.HostBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "host bindings are required")
	}

	return s.backend.UpsertHostBindings(ctx, req)
}

func (s *MockHostBindingServer) Delete(ctx context.Context, req *sgroupsv1.HostBindingReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host binding backend is not configured")
	}
	if req == nil || len(req.HostBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "host bindings are required")
	}
	if err := s.backend.DeleteHostBindings(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockHostBindingServer) List(ctx context.Context, req *sgroupsv1.HostBindingReq_List) (*sgroupsv1.HostBindingResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "host binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListHostBindings(ctx, req)
}

func (s *MockHostBindingServer) Watch(req *sgroupsv1.HostBindingReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.HostBindingResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "host binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchHostBindings(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockNetworkBindingServer) Upsert(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Upsert) (*sgroupsv1.NetworkBindingResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network binding backend is not configured")
	}
	if req == nil || len(req.NetworkBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "network bindings are required")
	}

	return s.backend.UpsertNetworkBindings(ctx, req)
}

func (s *MockNetworkBindingServer) Delete(ctx context.Context, req *sgroupsv1.NetworkBindingReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network binding backend is not configured")
	}
	if req == nil || len(req.NetworkBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "network bindings are required")
	}
	if err := s.backend.DeleteNetworkBindings(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockNetworkBindingServer) List(ctx context.Context, req *sgroupsv1.NetworkBindingReq_List) (*sgroupsv1.NetworkBindingResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "network binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListNetworkBindings(ctx, req)
}

func (s *MockNetworkBindingServer) Watch(req *sgroupsv1.NetworkBindingReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.NetworkBindingResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "network binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchNetworkBindings(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockServiceServer) Upsert(ctx context.Context, req *sgroupsv1.ServiceReq_Upsert) (*sgroupsv1.ServiceResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service backend is not configured")
	}
	if req == nil || len(req.Services) == 0 {
		return nil, status.Error(codes.InvalidArgument, "services are required")
	}

	return s.backend.UpsertServices(ctx, req)
}

func (s *MockServiceServer) Delete(ctx context.Context, req *sgroupsv1.ServiceReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service backend is not configured")
	}
	if req == nil || len(req.Services) == 0 {
		return nil, status.Error(codes.InvalidArgument, "services are required")
	}
	if err := s.backend.DeleteServices(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockServiceServer) List(ctx context.Context, req *sgroupsv1.ServiceReq_List) (*sgroupsv1.ServiceResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListServices(ctx, req)
}

func (s *MockServiceServer) Watch(req *sgroupsv1.ServiceReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.ServiceResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "service backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchServices(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func (s *MockServiceBindingServer) Upsert(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Upsert) (*sgroupsv1.ServiceBindingResp_Upsert, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service binding backend is not configured")
	}
	if req == nil || len(req.ServiceBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "service bindings are required")
	}

	return s.backend.UpsertServiceBindings(ctx, req)
}

func (s *MockServiceBindingServer) Delete(ctx context.Context, req *sgroupsv1.ServiceBindingReq_Delete) (*emptypb.Empty, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service binding backend is not configured")
	}
	if req == nil || len(req.ServiceBindings) == 0 {
		return nil, status.Error(codes.InvalidArgument, "service bindings are required")
	}
	if err := s.backend.DeleteServiceBindings(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *MockServiceBindingServer) List(ctx context.Context, req *sgroupsv1.ServiceBindingReq_List) (*sgroupsv1.ServiceBindingResp_List, error) {
	if s.backend == nil {
		return nil, status.Error(codes.Unavailable, "service binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return nil, status.Error(codes.InvalidArgument, "selectors are required")
	}

	return s.backend.ListServiceBindings(ctx, req)
}

func (s *MockServiceBindingServer) Watch(req *sgroupsv1.ServiceBindingReq_Watch, stream grpc.ServerStreamingServer[sgroupsv1.ServiceBindingResp_Watch]) error {
	if s.backend == nil {
		return status.Error(codes.Unavailable, "service binding backend is not configured")
	}
	if req == nil || len(req.Selectors) == 0 {
		return status.Error(codes.InvalidArgument, "selectors are required")
	}
	ws, err := s.backend.WatchServiceBindings(stream.Context(), req)
	if err != nil {
		return err
	}
	defer func() {
		if ws.Close != nil {
			ws.Close()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt, ok := <-ws.C:
			if !ok {
				return nil
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}
