package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Resource kind constants.
const (
	KindTenant     = "Tenant"
	KindTenantList = "TenantList"

	KindAddressGroup     = "AddressGroup"
	KindAddressGroupList = "AddressGroupList"

	KindNetwork     = "Network"
	KindNetworkList = "NetworkList"

	KindHost     = "Host"
	KindHostList = "HostList"

	KindSocketStat        = "SocketStat"
	KindSocketStatList    = "SocketStatList"
	KindSocketStatOptions = "SocketStatOptions"

	KindNft     = "Nft"
	KindNftList = "NftList"

	KindHostBinding     = "HostBinding"
	KindHostBindingList = "HostBindingList"

	KindNetworkBinding     = "NetworkBinding"
	KindNetworkBindingList = "NetworkBindingList"

	KindService     = "Service"
	KindServiceList = "ServiceList"

	KindServiceBinding     = "ServiceBinding"
	KindServiceBindingList = "ServiceBindingList"

	KindRule     = "Rule"
	KindRuleList = "RuleList"
)

// Resource plural name constants.
const (
	ResourceTenants         = "tenants"
	ResourceAddressGroups   = "addressgroups"
	ResourceNetworks        = "networks"
	ResourceHosts           = "hosts"
	ResourceHostBindings    = "hostbindings"
	ResourceNetworkBindings = "networkbindings"
	ResourceServices        = "services"
	ResourceServiceBindings = "servicebindings"
	ResourceRules           = "rules"

	// SubresourceHostSocketStats is the subresource name for hosts/sockstats.
	SubresourceHostSocketStats = "sockstats"

	// SubresourceHostNft is the subresource name for hosts/nft.
	SubresourceHostNft = "nft"
)

// Action represents the default action for an AddressGroup.
type Action string

const (
	ActionUnknown Action = "Unknown"
	ActionAllow   Action = "Allow"
	ActionDeny    Action = "Deny"
)

// TenantSpec defines the desired state of a Tenant.
type TenantSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Description string `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Tenant represents a tenant (namespace) resource.
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TenantSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// TenantList is a list of Tenant resources.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

// AddressGroupSpec defines the desired state of an AddressGroup.
type AddressGroupSpec struct {
	DisplayName   string `json:"displayName,omitempty"`
	Comment       string `json:"comment,omitempty"`
	Description   string `json:"description,omitempty"`
	DefaultAction Action `json:"defaultAction,omitempty"`
	Logs          bool   `json:"logs,omitempty"`
	Trace         bool   `json:"trace,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// AddressGroup represents an address group resource.
type AddressGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AddressGroupSpec `json:"spec,omitempty"`
	Refs              []ResourceRef    `json:"refs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// AddressGroupList is a list of AddressGroup resources.
type AddressGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddressGroup `json:"items"`
}

// NetworkSpec defines the desired state of a Network.
type NetworkSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Description string `json:"description,omitempty"`
	CIDR        string `json:"CIDR,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Network represents a network resource.
type Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkSpec   `json:"spec,omitempty"`
	Refs              []ResourceRef `json:"refs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NetworkList is a list of Network resources.
type NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Network `json:"items"`
}

// ResourceIdentifier identifies a resource by name and namespace.
type ResourceIdentifier struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// ResourceRef represents a read-only reference to a related resource.
type ResourceRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

// HostIPs contains IP addresses associated with a Host (read-only).
type HostIPs struct {
	IPv4 []string `json:"IPv4"`
	IPv6 []string `json:"IPv6"`
}

// HostMetaInfo contains system information about a Host (read-only).
type HostMetaInfo struct {
	HostName        string `json:"hostName"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformFamily  string `json:"platformFamily"`
	PlatformVersion string `json:"platformVersion"`
	KernelVersion   string `json:"kernelVersion"`
}

// HostEndpointPort is a named port exposed by a Host endpoint (read-only).
type HostEndpointPort struct {
	Name string `json:"name"`
	Port uint32 `json:"port"`
}

// HostEndpoints describes endpoints published by a Host (read-only).
// If Address is empty, the backend falls back to the peer address observed
// on the gRPC connection from the host agent.
type HostEndpoints struct {
	Address string             `json:"address"`
	Ports   []HostEndpointPort `json:"ports"`
}

// HostSpec defines the desired state of a Host.
type HostSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Description string `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Host represents a host resource.
type Host struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HostSpec      `json:"spec,omitempty"`
	Refs              []ResourceRef `json:"refs,omitempty"`
	IPs               HostIPs       `json:"ips,omitempty"`
	MetaInfo          HostMetaInfo  `json:"metaInfo,omitempty"`
	Endpoints         HostEndpoints `json:"endpoints,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HostList is a list of Host resources.
type HostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Host `json:"items"`
}

// HostBindingSpec defines the desired state of a HostBinding.
type HostBindingSpec struct {
	DisplayName  string             `json:"displayName,omitempty"`
	Comment      string             `json:"comment,omitempty"`
	Description  string             `json:"description,omitempty"`
	AddressGroup ResourceIdentifier `json:"addressGroup,omitempty"`
	Host         ResourceIdentifier `json:"host,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HostBinding represents a host binding resource.
type HostBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              HostBindingSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HostBindingList is a list of HostBinding resources.
type HostBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostBinding `json:"items"`
}

// NetworkBindingSpec defines the desired state of a NetworkBinding.
type NetworkBindingSpec struct {
	DisplayName  string             `json:"displayName,omitempty"`
	Comment      string             `json:"comment,omitempty"`
	Description  string             `json:"description,omitempty"`
	AddressGroup ResourceIdentifier `json:"addressGroup,omitempty"`
	Network      ResourceIdentifier `json:"network,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NetworkBinding represents a network binding resource.
type NetworkBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkBindingSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NetworkBindingList is a list of NetworkBinding resources.
type NetworkBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkBinding `json:"items"`
}

// ServiceTransportEntry represents a transport entry (ports or ICMP types) for a service.
type ServiceTransportEntry struct {
	Description string   `json:"description,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Ports       string   `json:"ports,omitempty"`
	Types       []uint32 `json:"types,omitempty"`
}

// ServiceTransport represents network transport configuration for a service.
type ServiceTransport struct {
	Protocol Protocol                `json:"protocol,omitempty"`
	IPv      IpAddrFamily            `json:"IPv,omitempty"`
	Entries  []ServiceTransportEntry `json:"entries,omitempty"`
}

// ServiceSpec defines the desired state of a Service.
type ServiceSpec struct {
	DisplayName string             `json:"displayName,omitempty"`
	Comment     string             `json:"comment,omitempty"`
	Description string             `json:"description,omitempty"`
	Transports  []ServiceTransport `json:"transports,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Service represents a service resource.
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceSpec   `json:"spec,omitempty"`
	Refs              []ResourceRef `json:"refs,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ServiceList is a list of Service resources.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

// ServiceBindingSpec defines the desired state of a ServiceBinding.
type ServiceBindingSpec struct {
	DisplayName  string             `json:"displayName,omitempty"`
	Comment      string             `json:"comment,omitempty"`
	Description  string             `json:"description,omitempty"`
	AddressGroup ResourceIdentifier `json:"addressGroup,omitempty"`
	Service      ResourceIdentifier `json:"service,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ServiceBinding represents a service binding resource.
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceBindingSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ServiceBindingList is a list of ServiceBinding resources.
type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

// Traffic represents traffic direction.
type Traffic string

const (
	TrafficBoth    Traffic = "Both"
	TrafficIngress Traffic = "Ingress"
	TrafficEgress  Traffic = "Egress"
)

// Protocol represents network transport protocol.
type Protocol string

const (
	ProtocolTCP  Protocol = "TCP"
	ProtocolUDP  Protocol = "UDP"
	ProtocolICMP Protocol = "ICMP"
)

// IpAddrFamily represents IP address family.
type IpAddrFamily string

const (
	IpAddrFamilyUndef IpAddrFamily = ""
	IpAddrFamilyIPv4  IpAddrFamily = "IPv4"
	IpAddrFamilyIPv6  IpAddrFamily = "IPv6"
)

// EndpointType represents endpoint type.
type EndpointType string

const (
	EndpointTypeUnknown      EndpointType = "Unknown"
	EndpointTypeAddressGroup EndpointType = "AddressGroup"
	EndpointTypeService      EndpointType = "Service"
	EndpointTypeFQDN         EndpointType = "FQDN"
	EndpointTypeCIDR         EndpointType = "CIDR"
)

// RuleEndpoint represents a local or remote endpoint.
type RuleEndpoint struct {
	Name      string            `json:"name,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Type      EndpointType      `json:"type,omitempty"`
	Value     string            `json:"value,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// RuleEndpoints represents local and remote endpoints.
type RuleEndpoints struct {
	Local  *RuleEndpoint `json:"local,omitempty"`
	Remote *RuleEndpoint `json:"remote,omitempty"`
}

// TransportEntry represents a transport entry (ports or ICMP types).
type TransportEntry struct {
	Description string   `json:"description,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Ports       string   `json:"ports,omitempty"`
	Types       []uint32 `json:"types,omitempty"`
}

// RuleTransport represents network transport configuration.
type RuleTransport struct {
	Protocol Protocol         `json:"protocol,omitempty"`
	IPv      IpAddrFamily     `json:"IPv,omitempty"`
	Entries  []TransportEntry `json:"entries,omitempty"`
}

// RuleSession represents session parameters.
type RuleSession struct {
	Traffic Traffic `json:"traffic,omitempty"`
}

// RuleSpec defines the desired state of a Rule.
type RuleSpec struct {
	DisplayName string         `json:"displayName,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	Description string         `json:"description,omitempty"`
	Action      Action         `json:"action,omitempty"`
	Session     *RuleSession   `json:"session,omitempty"`
	Endpoints   *RuleEndpoints `json:"endpoints,omitempty"`
	Transport   *RuleTransport `json:"transport,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Rule represents a rule resource.
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RuleSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RuleList is a list of Rule resources.
type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rule `json:"items"`
}

type ConnState string

const (
	ConnStateUnknown     ConnState = "Unknown"
	ConnStateEstablished ConnState = "Established"
	ConnStateSynSent     ConnState = "SynSent"
	ConnStateSynRecv     ConnState = "SynRecv"
	ConnStateFinWait1    ConnState = "FinWait1"
	ConnStateFinWait2    ConnState = "FinWait2"
	ConnStateTimeWait    ConnState = "TimeWait"
	ConnStateClose       ConnState = "Close"
	ConnStateCloseWait   ConnState = "CloseWait"
	ConnStateLastAck     ConnState = "LastAck"
	ConnStateListen      ConnState = "Listen"
	ConnStateClosing     ConnState = "Closing"
	ConnStateNewSynRecv  ConnState = "NewSynRecv"
)

// Process represents a kernel-reported process that owns a socket.
type Process struct {
	PID     int32  `json:"pid"`
	Comm    string `json:"comm,omitempty"`
	CmdLine string `json:"cmdLine,omitempty"`
	Exe     string `json:"exe,omitempty"`
	FD      int32  `json:"fd,omitempty"`
}

// SocketStatSelector is a single-statistic filter accepted by the subresource.
// Empty fields are wildcards. Multiple selectors in a request are OR-joined.
type SocketStatSelector struct {
	Protocol   string       `json:"protocol,omitempty"`
	Family     IpAddrFamily `json:"family,omitempty"`
	State      ConnState    `json:"state,omitempty"`
	LocalAddr  string       `json:"localAddr,omitempty"`
	LocalPort  int32        `json:"localPort,omitempty"`
	RemoteAddr string       `json:"remoteAddr,omitempty"`
	RemotePort int32        `json:"remotePort,omitempty"`
	Ifname     string       `json:"ifname,omitempty"`
	Inode      int64        `json:"inode,omitempty"`
	PID        int32        `json:"pid,omitempty"`
	Comm       string       `json:"comm,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// SocketStatOptions is the body/query payload for the sockstats subresource.
// Watch toggles between list and watch behavior — same URL pattern as Pod log.
type SocketStatOptions struct {
	metav1.TypeMeta `json:",inline"`
	// Watch turns the request into a stream of SocketStatList chunks.
	Watch bool `json:"watch,omitempty"`
	// Selectors are OR-joined filters; empty means "all".
	Selectors []SocketStatSelector `json:"selectors,omitempty"`
}

// SocketStat is a single socket-statistics entry returned by a host-agent.
type SocketStat struct {
	Protocol   string       `json:"protocol,omitempty"`
	Family     IpAddrFamily `json:"family,omitempty"`
	State      ConnState    `json:"state,omitempty"`
	LocalAddr  string       `json:"localAddr,omitempty"`
	LocalPort  int32        `json:"localPort,omitempty"`
	RemoteAddr string       `json:"remoteAddr,omitempty"`
	RemotePort int32        `json:"remotePort,omitempty"`
	Ifname     string       `json:"ifname,omitempty"`
	Inode      int64        `json:"inode,omitempty"`
	Processes  []Process    `json:"processes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// SocketStatList is the list response for the sockstats subresource.
// Also used as the per-event payload for watch (one event = batch of stats).
type SocketStatList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Host            ResourceIdentifier `json:"host,omitempty"`
	Items           []SocketStat       `json:"items"`
}

// NftOptions is the hosts/nft query payload; Watch is the only knob (no selectors).
type NftOptions struct {
	Watch bool `json:"watch,omitempty"`
}

// Nft is one host's nftables ruleset, as reported by the host-agent.
type Nft struct {
	// Text is the `nft list ruleset` output.
	Text string `json:"text,omitempty"`
	// JSON is the `nft -j` ruleset as structured JSON; pointer so it omits
	// (not "json": null) when no parseable JSON is returned.
	JSON *runtime.RawExtension `json:"json,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NftList is the hosts/nft list response (and per-event watch payload).
type NftList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Host            ResourceIdentifier `json:"host,omitempty"`
	Items           []Nft              `json:"items"`
}

// ---------- OpenAPIModelName ----------
// The Kubernetes DefinitionNamer converts Go import paths (slashes) to
// dot-separated names. Types must implement OpenAPIModelName to match,
// otherwise $ref pointers contain JSON-Pointer-escaped slashes (~1)
// that the field manager cannot resolve.

// OpenAPIPrefix is the dot-separated prefix for OpenAPI model names.
// Exported for use by hack/openapi-spec.
const OpenAPIPrefix = "sgroups.io.sgroups-k8s-api.pkg.apis.sgroups.v1alpha1."

func (Tenant) OpenAPIModelName() string             { return OpenAPIPrefix + "Tenant" }
func (TenantList) OpenAPIModelName() string         { return OpenAPIPrefix + "TenantList" }
func (TenantSpec) OpenAPIModelName() string         { return OpenAPIPrefix + "TenantSpec" }
func (AddressGroup) OpenAPIModelName() string       { return OpenAPIPrefix + "AddressGroup" }
func (AddressGroupList) OpenAPIModelName() string   { return OpenAPIPrefix + "AddressGroupList" }
func (AddressGroupSpec) OpenAPIModelName() string   { return OpenAPIPrefix + "AddressGroupSpec" }
func (Network) OpenAPIModelName() string            { return OpenAPIPrefix + "Network" }
func (NetworkList) OpenAPIModelName() string        { return OpenAPIPrefix + "NetworkList" }
func (NetworkSpec) OpenAPIModelName() string        { return OpenAPIPrefix + "NetworkSpec" }
func (Host) OpenAPIModelName() string               { return OpenAPIPrefix + "Host" }
func (HostList) OpenAPIModelName() string           { return OpenAPIPrefix + "HostList" }
func (HostSpec) OpenAPIModelName() string           { return OpenAPIPrefix + "HostSpec" }
func (HostBinding) OpenAPIModelName() string        { return OpenAPIPrefix + "HostBinding" }
func (HostBindingList) OpenAPIModelName() string    { return OpenAPIPrefix + "HostBindingList" }
func (HostBindingSpec) OpenAPIModelName() string    { return OpenAPIPrefix + "HostBindingSpec" }
func (NetworkBinding) OpenAPIModelName() string     { return OpenAPIPrefix + "NetworkBinding" }
func (NetworkBindingList) OpenAPIModelName() string { return OpenAPIPrefix + "NetworkBindingList" }
func (NetworkBindingSpec) OpenAPIModelName() string { return OpenAPIPrefix + "NetworkBindingSpec" }
func (ResourceIdentifier) OpenAPIModelName() string { return OpenAPIPrefix + "ResourceIdentifier" }
func (ResourceRef) OpenAPIModelName() string        { return OpenAPIPrefix + "ResourceRef" }
func (HostIPs) OpenAPIModelName() string            { return OpenAPIPrefix + "HostIPs" }
func (HostEndpoints) OpenAPIModelName() string      { return OpenAPIPrefix + "HostEndpoints" }
func (HostEndpointPort) OpenAPIModelName() string   { return OpenAPIPrefix + "HostEndpointPort" }
func (HostMetaInfo) OpenAPIModelName() string       { return OpenAPIPrefix + "HostMetaInfo" }
func (Service) OpenAPIModelName() string            { return OpenAPIPrefix + "Service" }
func (ServiceList) OpenAPIModelName() string        { return OpenAPIPrefix + "ServiceList" }
func (ServiceSpec) OpenAPIModelName() string        { return OpenAPIPrefix + "ServiceSpec" }
func (ServiceTransport) OpenAPIModelName() string   { return OpenAPIPrefix + "ServiceTransport" }
func (ServiceTransportEntry) OpenAPIModelName() string {
	return OpenAPIPrefix + "ServiceTransportEntry"
}
func (ServiceBinding) OpenAPIModelName() string     { return OpenAPIPrefix + "ServiceBinding" }
func (ServiceBindingList) OpenAPIModelName() string { return OpenAPIPrefix + "ServiceBindingList" }
func (ServiceBindingSpec) OpenAPIModelName() string { return OpenAPIPrefix + "ServiceBindingSpec" }
func (Rule) OpenAPIModelName() string               { return OpenAPIPrefix + "Rule" }
func (RuleList) OpenAPIModelName() string           { return OpenAPIPrefix + "RuleList" }
func (RuleSpec) OpenAPIModelName() string           { return OpenAPIPrefix + "RuleSpec" }
func (RuleEndpoint) OpenAPIModelName() string       { return OpenAPIPrefix + "RuleEndpoint" }
func (RuleEndpoints) OpenAPIModelName() string      { return OpenAPIPrefix + "RuleEndpoints" }
func (RuleSession) OpenAPIModelName() string        { return OpenAPIPrefix + "RuleSession" }
func (RuleTransport) OpenAPIModelName() string      { return OpenAPIPrefix + "RuleTransport" }
func (TransportEntry) OpenAPIModelName() string     { return OpenAPIPrefix + "TransportEntry" }
func (SocketStat) OpenAPIModelName() string         { return OpenAPIPrefix + "SocketStat" }
func (SocketStatList) OpenAPIModelName() string     { return OpenAPIPrefix + "SocketStatList" }
func (SocketStatOptions) OpenAPIModelName() string  { return OpenAPIPrefix + "SocketStatOptions" }
func (SocketStatSelector) OpenAPIModelName() string { return OpenAPIPrefix + "SocketStatSelector" }
func (Process) OpenAPIModelName() string            { return OpenAPIPrefix + "Process" }
func (Nft) OpenAPIModelName() string                { return OpenAPIPrefix + "Nft" }
func (NftList) OpenAPIModelName() string            { return OpenAPIPrefix + "NftList" }
