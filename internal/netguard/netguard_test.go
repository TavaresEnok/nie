package netguard

import (
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2001:4860:4860::8888", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"10.255.255.255", false},
		{"172.16.0.1", false},
		{"172.31.255.255", false},
		{"192.168.1.1", false},
		{"100.64.0.1", false},
		{"100.127.255.255", false},
		{"169.254.169.254", false},
		{"192.0.0.1", false},
		{"192.0.2.1", false},
		{"198.18.0.1", false},
		{"198.51.100.1", false},
		{"203.0.113.1", false},
		{"224.0.0.1", false},
		{"255.255.255.255", false},
		{"0.0.0.0", false},
		{"::1", false},
		{"fc00::1", false},
		{"fd00:ec2::254", false},
		{"fe80::1", false},
		{"2001:db8::1", false},
		{"::ffff:127.0.0.1", false},
		{"::", false},
	}
	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if ip == nil {
			t.Fatalf("IP de teste inválido: %s", tc.ip)
		}
		if got := IsPublicIP(ip); got != tc.want {
			t.Errorf("IsPublicIP(%s) = %v, quer %v", tc.ip, got, tc.want)
		}
	}
	if IsPublicIP(nil) {
		t.Error("IsPublicIP(nil) deve ser false")
	}
}
