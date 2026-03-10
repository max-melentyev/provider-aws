package ec2

import (
	"testing"
)

func TestCIDRBlocksEqual(t *testing.T) {
	tests := []struct {
		name  string
		cidr1 string
		cidr2 string
		want  bool
	}{
		{
			name:  "identical IPv4 CIDRs",
			cidr1: "10.0.0.0/16",
			cidr2: "10.0.0.0/16",
			want:  true,
		},
		{
			name:  "different IPv4 CIDRs",
			cidr1: "10.0.0.0/16",
			cidr2: "10.1.0.0/16",
			want:  false,
		},
		{
			name:  "identical IPv6 CIDRs",
			cidr1: "2604:2be0::/32",
			cidr2: "2604:2be0::/32",
			want:  true,
		},
		{
			name:  "equivalent IPv6 CIDRs with expanded zeros",
			cidr1: "2604:2be0:0000::/32",
			cidr2: "2604:2be0::/32",
			want:  true,
		},
		{
			name:  "different IPv6 CIDRs",
			cidr1: "2604:2be0::/32",
			cidr2: "2604:2be1::/32",
			want:  false,
		},
		{
			name:  "same IPv6 network different prefix length",
			cidr1: "2604:2be0::/32",
			cidr2: "2604:2be0::/48",
			want:  false,
		},
		{
			name:  "empty strings",
			cidr1: "",
			cidr2: "",
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CIDRBlocksEqual(tc.cidr1, tc.cidr2)
			if got != tc.want {
				t.Errorf("CIDRBlocksEqual(%q, %q) = %v, want %v", tc.cidr1, tc.cidr2, got, tc.want)
			}
		})
	}
}
