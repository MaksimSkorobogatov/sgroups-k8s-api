package v1alpha1

import "fmt"

// Validate returns an error if a is not one of the well-known Action constants.
// Empty value is treated as unset and accepted (caller decides if required).
func (a Action) Validate() error {
	switch a {
	case "", ActionAllow, ActionDeny:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q] (case-sensitive), got %q",
			ActionAllow, ActionDeny, a)
	}
}

// Validate returns an error if t is not one of the well-known Traffic constants.
// Empty value is treated as unset and accepted (caller decides if required).
func (t Traffic) Validate() error {
	switch t {
	case "", TrafficBoth, TrafficIngress, TrafficEgress:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q, %q] (case-sensitive), got %q",
			TrafficBoth, TrafficIngress, TrafficEgress, t)
	}
}

// Validate returns an error if p is not one of the well-known Protocol constants.
// Empty value is treated as unset and accepted (caller decides if required).
func (p Protocol) Validate() error {
	switch p {
	case "", ProtocolTCP, ProtocolUDP, ProtocolICMP:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q, %q] (case-sensitive), got %q",
			ProtocolTCP, ProtocolUDP, ProtocolICMP, p)
	}
}

// Validate returns an error if f is not one of the well-known IpAddrFamily constants.
func (f IpAddrFamily) Validate() error {
	switch f {
	case IpAddrFamilyUndef, IpAddrFamilyIPv4, IpAddrFamilyIPv6:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q] (case-sensitive), got %q",
			IpAddrFamilyIPv4, IpAddrFamilyIPv6, f)
	}
}

// Validate returns an error if e is not one of the well-known EndpointType constants.
// Empty value is treated as unset and accepted (caller decides if required).
func (e EndpointType) Validate() error {
	switch e {
	case "", EndpointTypeAddressGroup, EndpointTypeService, EndpointTypeFQDN, EndpointTypeCIDR:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q, %q, %q] (case-sensitive), got %q",
			EndpointTypeAddressGroup, EndpointTypeService, EndpointTypeFQDN, EndpointTypeCIDR, e)
	}
}

// Validate returns an error if s is not one of the well-known ConnState constants.
// Empty / "Unknown" are treated as unset (accepted — caller decides if required).
func (s ConnState) Validate() error {
	switch s {
	case "", ConnStateUnknown,
		ConnStateEstablished, ConnStateSynSent, ConnStateSynRecv,
		ConnStateFinWait1, ConnStateFinWait2, ConnStateTimeWait,
		ConnStateClose, ConnStateCloseWait, ConnStateLastAck,
		ConnStateListen, ConnStateClosing, ConnStateNewSynRecv:
		return nil
	default:
		return fmt.Errorf("must be one of [%q, %q, %q, %q, %q, %q, %q, %q, %q, %q, %q, %q] (case-sensitive), got %q",
			ConnStateEstablished, ConnStateSynSent, ConnStateSynRecv,
			ConnStateFinWait1, ConnStateFinWait2, ConnStateTimeWait,
			ConnStateClose, ConnStateCloseWait, ConnStateLastAck,
			ConnStateListen, ConnStateClosing, ConnStateNewSynRecv, s)
	}
}
