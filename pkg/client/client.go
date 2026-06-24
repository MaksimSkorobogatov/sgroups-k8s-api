package client

import (
	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Conn            *grpc.ClientConn
	Namespaces      sgroupsv1.SGroupsNamespaceAPIClient
	AddressGroups   sgroupsv1.SGroupsAddressGroupsAPIClient
	Networks        sgroupsv1.SGroupsNetworksAPIClient
	Hosts           sgroupsv1.SGroupsHostsAPIClient
	HostBindings    sgroupsv1.SGroupsHostBindingAPIClient
	NetworkBindings sgroupsv1.SGroupsNetworkBindingAPIClient
	Services        sgroupsv1.SGroupsServicesAPIClient
	ServiceBindings sgroupsv1.SGroupsServiceBindingAPIClient
	Rules           sgroupsv1.SGroupsRulesAPIClient
	Authn           sgroupsv1.SGroupsAuthnAPIClient
}

func Dial(addr string, opts ...grpc.DialOption) (*Client, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	c := newClient(conn)
	c.Authn = sgroupsv1.NewSGroupsAuthnAPIClient(conn)

	return c, nil
}

func (c *Client) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}

	return c.Conn.Close()
}
