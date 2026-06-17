package tenant

import "strings"

var reservedExactNames = map[string]struct{}{
	"default":         {},
	"kube-system":     {},
	"kube-public":     {},
	"kube-node-lease": {},
	"sgroups-system":  {},
}

var reservedPrefixes = []string{
	"kube-",
	"sgroups-",
}

// IsReservedName reports whether name is forbidden as a Tenant.
func IsReservedName(name string) bool {
	if _, ok := reservedExactNames[name]; ok {
		return true
	}
	for _, p := range reservedPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}

	return false
}

// ReservedReason returns a short explanation, or "" if name is not reserved.
func ReservedReason(name string) string {
	if _, ok := reservedExactNames[name]; ok {
		return "name is reserved for the platform"
	}
	for _, p := range reservedPrefixes {
		if strings.HasPrefix(name, p) {
			return "names with prefix " + p + " are reserved"
		}
	}

	return ""
}
