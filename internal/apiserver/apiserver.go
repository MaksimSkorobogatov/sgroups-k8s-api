package apiserver

import (
	"errors"

	"k8s.io/apiserver/pkg/server"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// CompletedConfig is the completed server configuration.
type CompletedConfig struct {
	GenericConfig server.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// Complete completes the config.
func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{
		GenericConfig: c.GenericConfig.Complete(),
		ExtraConfig:   &c.ExtraConfig,
	}
}

// Server wraps the generic API server.
type Server struct {
	GenericAPIServer *server.GenericAPIServer
}

// New creates a new SGroups aggregated API server.
func (c CompletedConfig) New() (*Server, error) {
	if c.ExtraConfig == nil || c.ExtraConfig.GRPCClient == nil {
		return nil, errors.New("grpc client is required")
	}

	genericServer, err := c.GenericConfig.New("sgroups-k8s-apiserver", server.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	apiGroupInfo := server.NewDefaultAPIGroupInfo(
		v1alpha1.GroupName,
		v1alpha1.Scheme,
		v1alpha1.ParameterCodec,
		v1alpha1.Codecs,
	)

	apiGroupInfo.VersionedResourcesStorageMap = buildStorageMap(c.ExtraConfig.GRPCClient, c.ExtraConfig.StorageOptions)
	installSubresources(apiGroupInfo.VersionedResourcesStorageMap, c.ExtraConfig.GRPCClient, c.ExtraConfig.StorageOptions)

	if err := genericServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return &Server{GenericAPIServer: genericServer}, nil
}
