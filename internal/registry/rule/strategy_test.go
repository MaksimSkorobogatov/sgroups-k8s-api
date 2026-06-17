package rule

import (
	"context"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

// rule that mirrors a realistic AG→AG ingress rule with TCP/IPv4 transport.
func validRule() *v1alpha1.Rule {
	return &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "ns"},
		Spec: v1alpha1.RuleSpec{
			Action:  v1alpha1.ActionAllow,
			Session: &v1alpha1.RuleSession{Traffic: v1alpha1.TrafficIngress},
			Endpoints: &v1alpha1.RuleEndpoints{
				Local:  &v1alpha1.RuleEndpoint{Type: v1alpha1.EndpointTypeAddressGroup, Name: "a", Namespace: "ns"},
				Remote: &v1alpha1.RuleEndpoint{Type: v1alpha1.EndpointTypeAddressGroup, Name: "b", Namespace: "ns"},
			},
			Transport: &v1alpha1.RuleTransport{
				Protocol: v1alpha1.ProtocolTCP,
				IPv:      v1alpha1.IpAddrFamilyIPv4,
				Entries:  []v1alpha1.TransportEntry{{Description: "HTTP", Ports: "80"}},
			},
		},
	}
}

func TestRuleStrategy_AcceptsValid(t *testing.T) {
	t.Parallel()
	s := &ruleStrategy{}
	if err := s.ValidateCreate(context.Background(), validRule()); err != nil {
		t.Fatalf("ValidateCreate(valid): %v", err)
	}
	if err := s.ValidateUpdate(context.Background(), validRule(), validRule()); err != nil {
		t.Fatalf("ValidateUpdate(valid): %v", err)
	}
}

// The exact regression Pavel hit: lowercase 'ingress' must produce an
// actionable error pointing at spec.session.traffic instead of being
// silently dropped to TRAFFIC_UNDEF in the converter.
func TestRuleStrategy_RejectsLowercaseTraffic(t *testing.T) {
	t.Parallel()
	r := validRule()
	r.Spec.Session.Traffic = "ingress"
	err := (&ruleStrategy{}).ValidateCreate(context.Background(), r)
	if err == nil {
		t.Fatal("want error for lowercase 'ingress', got nil")
	}
	if !apierrors.IsBadRequest(err) {
		t.Errorf("want BadRequest, got %T: %v", err, err)
	}
	msg := err.Error()
	for _, want := range []string{"spec.session.traffic", "ingress", "case-sensitive"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error must contain %q; got: %s", want, msg)
		}
	}
}

func TestRuleStrategy_RejectsEachEnumField(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		mutate   func(*v1alpha1.Rule)
		wantPath string
	}{
		{"action", func(r *v1alpha1.Rule) { r.Spec.Action = "allow" }, "spec.action"},
		{"local.type", func(r *v1alpha1.Rule) { r.Spec.Endpoints.Local.Type = "addressgroup" }, "spec.endpoints.local.type"},
		{"remote.type", func(r *v1alpha1.Rule) { r.Spec.Endpoints.Remote.Type = "service" }, "spec.endpoints.remote.type"},
		{"transport.protocol", func(r *v1alpha1.Rule) { r.Spec.Transport.Protocol = "tcp" }, "spec.transport.protocol"},
		{"transport.IPv", func(r *v1alpha1.Rule) { r.Spec.Transport.IPv = "ipv4" }, "spec.transport.IPv"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRule()
			tc.mutate(r)
			err := (&ruleStrategy{}).ValidateCreate(context.Background(), r)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantPath) {
				t.Errorf("error must point at %q; got: %s", tc.wantPath, err.Error())
			}
		})
	}
}

// Empty/nil sub-structs must not trip validation — they're omitempty fields
// the user simply didn't set, and required-ness is the backend's call.
func TestRuleStrategy_TolerantOfNilSubstructs(t *testing.T) {
	t.Parallel()
	r := &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal", Namespace: "ns"},
		Spec:       v1alpha1.RuleSpec{Action: v1alpha1.ActionAllow},
	}
	if err := (&ruleStrategy{}).ValidateCreate(context.Background(), r); err != nil {
		t.Fatalf("minimal Rule: %v", err)
	}
}
