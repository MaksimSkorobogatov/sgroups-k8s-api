package v1alpha1

import (
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// GetEnumOpenAPIDefinitions returns OpenAPI definitions with enum support for custom types.
func GetEnumOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		OpenAPIPrefix + "Action": {
			Schema: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "Action (Allow or Deny)",
					Type:        []string{"string"},
					Enum: []interface{}{
						string(ActionAllow),
						string(ActionDeny),
					},
				},
			},
		},
		OpenAPIPrefix + "Protocol": {
			Schema: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "Network transport protocol (TCP, UDP, or ICMP)",
					Type:        []string{"string"},
					Enum: []interface{}{
						string(ProtocolTCP),
						string(ProtocolUDP),
						string(ProtocolICMP),
					},
				},
			},
		},
		OpenAPIPrefix + "IpAddrFamily": {
			Schema: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "IP address family (IPv4 or IPv6)",
					Type:        []string{"string"},
					Enum: []interface{}{
						string(IpAddrFamilyIPv4),
						string(IpAddrFamilyIPv6),
					},
				},
			},
		},
		OpenAPIPrefix + "Traffic": {
			Schema: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "Traffic direction (Both, Ingress, or Egress)",
					Type:        []string{"string"},
					Enum: []interface{}{
						string(TrafficBoth),
						string(TrafficIngress),
						string(TrafficEgress),
					},
				},
			},
		},
		OpenAPIPrefix + "EndpointType": {
			Schema: spec.Schema{
				SchemaProps: spec.SchemaProps{
					Description: "Endpoint type (AddressGroup, Service, FQDN, or CIDR)",
					Type:        []string{"string"},
					Enum: []interface{}{
						string(EndpointTypeAddressGroup),
						string(EndpointTypeService),
						string(EndpointTypeFQDN),
						string(EndpointTypeCIDR),
					},
				},
			},
		},
	}
}

// GetOpenAPIDefinitionsWithEnums returns OpenAPI definitions including enum metadata.
func GetOpenAPIDefinitionsWithEnums(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	defs := GetOpenAPIDefinitions(ref)
	for key, value := range GetEnumOpenAPIDefinitions(ref) {
		defs[key] = value
	}

	modifyStructFieldsWithEnums(defs)

	return defs
}

func modifyStructFieldsWithEnums(defs map[string]common.OpenAPIDefinition) {
	actionEnum := []interface{}{
		string(ActionAllow),
		string(ActionDeny),
	}
	protocolEnum := []interface{}{
		string(ProtocolTCP),
		string(ProtocolUDP),
		string(ProtocolICMP),
	}
	ipvEnum := []interface{}{
		string(IpAddrFamilyIPv4),
		string(IpAddrFamilyIPv6),
	}
	trafficEnum := []interface{}{
		string(TrafficBoth),
		string(TrafficIngress),
		string(TrafficEgress),
	}
	endpointTypeEnum := []interface{}{
		string(EndpointTypeAddressGroup),
		string(EndpointTypeService),
		string(EndpointTypeFQDN),
		string(EndpointTypeCIDR),
	}
	connStateEnum := []interface{}{
		string(ConnStateEstablished), string(ConnStateSynSent), string(ConnStateSynRecv),
		string(ConnStateFinWait1), string(ConnStateFinWait2), string(ConnStateTimeWait),
		string(ConnStateClose), string(ConnStateCloseWait), string(ConnStateLastAck),
		string(ConnStateListen), string(ConnStateClosing), string(ConnStateNewSynRecv),
	}

	setFieldEnum(defs, AddressGroupSpec{}.OpenAPIModelName(), "defaultAction", actionEnum)
	setFieldEnum(defs, ServiceTransport{}.OpenAPIModelName(), "protocol", protocolEnum)
	setFieldEnum(defs, ServiceTransport{}.OpenAPIModelName(), "IPv", ipvEnum)
	setFieldEnum(defs, RuleTransport{}.OpenAPIModelName(), "protocol", protocolEnum)
	setFieldEnum(defs, RuleTransport{}.OpenAPIModelName(), "IPv", ipvEnum)
	setFieldEnum(defs, RuleSpec{}.OpenAPIModelName(), "action", actionEnum)
	setFieldEnum(defs, RuleSession{}.OpenAPIModelName(), "traffic", trafficEnum)
	setFieldEnum(defs, RuleEndpoint{}.OpenAPIModelName(), "type", endpointTypeEnum)
	setFieldEnum(defs, SocketStat{}.OpenAPIModelName(), "state", connStateEnum)
	setFieldEnum(defs, SocketStat{}.OpenAPIModelName(), "family", ipvEnum)
}

func setFieldEnum(defs map[string]common.OpenAPIDefinition, defKey, fieldName string, enum []interface{}) {
	def, exists := defs[defKey]
	if !exists || def.Schema.Properties == nil {
		return
	}
	prop, ok := def.Schema.Properties[fieldName]
	if !ok {
		return
	}
	prop.Enum = enum
	def.Schema.Properties[fieldName] = prop
	defs[defKey] = def
}
