package tenant

import "testing"

func TestIsReservedName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"plain", "acme", false},
		{"single-letter", "a", false},
		{"hyphenated-business-name", "acme-corp", false},
		{"empty", "", false},

		{"exact default", "default", true},
		{"exact kube-system", "kube-system", true},
		{"exact kube-public", "kube-public", true},
		{"exact kube-node-lease", "kube-node-lease", true},
		{"exact sgroups-system", "sgroups-system", true},

		{"kube-prefix only", "kube-", true},
		{"kube-prefix custom", "kube-something", true},
		{"sgroups-prefix only", "sgroups-", true},
		{"sgroups-prefix custom", "sgroups-foo", true},

		// Names that look reserved but are not — substring is not enough.
		{"contains kube- inside", "my-kube-cluster", false},
		{"contains sgroups- inside", "x-sgroups-y", false},

		// Names that almost match exact but don't quite.
		{"defaultx", "defaultx", false},
		{"kubesystem (no hyphen)", "kubesystem", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsReservedName(tc.in); got != tc.want {
				t.Errorf("IsReservedName(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestReservedReason_NonEmptyForReserved(t *testing.T) {
	t.Parallel()

	for _, n := range []string{"default", "kube-system", "kube-foo", "sgroups-system", "sgroups-foo"} {
		if got := ReservedReason(n); got == "" {
			t.Errorf("ReservedReason(%q) returned empty string for reserved name", n)
		}
	}
	if got := ReservedReason("acme"); got != "" {
		t.Errorf("ReservedReason(\"acme\") = %q, want empty string", got)
	}
}
