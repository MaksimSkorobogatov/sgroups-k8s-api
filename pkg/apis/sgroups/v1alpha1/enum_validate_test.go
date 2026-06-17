package v1alpha1_test

import (
	"strings"
	"testing"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestTrafficValidate(t *testing.T) {
	t.Parallel()

	// Valid: documented constants + empty (omitempty support).
	for _, v := range []v1alpha1.Traffic{"", v1alpha1.TrafficBoth, v1alpha1.TrafficIngress, v1alpha1.TrafficEgress} {
		if err := v.Validate(); err != nil {
			t.Errorf("Traffic(%q): want valid, got %v", v, err)
		}
	}

	// Invalid: lowercase variant (Pavel's case — was silently dropped, now must error).
	for _, v := range []v1alpha1.Traffic{"ingress", "egress", "both", "INGRESS", "Other"} {
		err := v.Validate()
		if err == nil {
			t.Errorf("Traffic(%q): want error, got nil", v)
			continue
		}
		// Must be actionable: include the bad value so the user sees the typo.
		if !strings.Contains(err.Error(), string(v)) {
			t.Errorf("Traffic(%q): error must include the bad value, got: %v", v, err)
		}
		if !strings.Contains(err.Error(), "case-sensitive") {
			t.Errorf("Traffic(%q): error should advertise case-sensitivity, got: %v", v, err)
		}
	}
}

func TestActionValidate(t *testing.T) {
	t.Parallel()

	for _, v := range []v1alpha1.Action{"", v1alpha1.ActionAllow, v1alpha1.ActionDeny} {
		if err := v.Validate(); err != nil {
			t.Errorf("Action(%q): want valid, got %v", v, err)
		}
	}
	for _, v := range []v1alpha1.Action{"allow", "deny", "Allow ", "Other"} {
		if err := v.Validate(); err == nil {
			t.Errorf("Action(%q): want error, got nil", v)
		}
	}
}

func TestProtocolValidate(t *testing.T) {
	t.Parallel()

	for _, v := range []v1alpha1.Protocol{"", v1alpha1.ProtocolTCP, v1alpha1.ProtocolUDP, v1alpha1.ProtocolICMP} {
		if err := v.Validate(); err != nil {
			t.Errorf("Protocol(%q): want valid, got %v", v, err)
		}
	}
	for _, v := range []v1alpha1.Protocol{"tcp", "udp", "Tcp", "Other"} {
		if err := v.Validate(); err == nil {
			t.Errorf("Protocol(%q): want error, got nil", v)
		}
	}
}

func TestIpAddrFamilyValidate(t *testing.T) {
	t.Parallel()

	for _, v := range []v1alpha1.IpAddrFamily{v1alpha1.IpAddrFamilyUndef, v1alpha1.IpAddrFamilyIPv4, v1alpha1.IpAddrFamilyIPv6} {
		if err := v.Validate(); err != nil {
			t.Errorf("IpAddrFamily(%q): want valid, got %v", v, err)
		}
	}
	for _, v := range []v1alpha1.IpAddrFamily{"ipv4", "ipv6", "IPv47", "Other"} {
		if err := v.Validate(); err == nil {
			t.Errorf("IpAddrFamily(%q): want error, got nil", v)
		}
	}
}

func TestEndpointTypeValidate(t *testing.T) {
	t.Parallel()

	for _, v := range []v1alpha1.EndpointType{
		"",
		v1alpha1.EndpointTypeAddressGroup,
		v1alpha1.EndpointTypeService,
		v1alpha1.EndpointTypeFQDN,
		v1alpha1.EndpointTypeCIDR,
	} {
		if err := v.Validate(); err != nil {
			t.Errorf("EndpointType(%q): want valid, got %v", v, err)
		}
	}
	for _, v := range []v1alpha1.EndpointType{"addressgroup", "service", "fqdn", "cidr", "Unknown"} {
		if err := v.Validate(); err == nil {
			t.Errorf("EndpointType(%q): want error, got nil", v)
		}
	}
}
