// Package netguard concentra a política de egress usada pelo crawler e pela
// validação de URLs. Ele rejeita destinos que não podem ser alcançados por um
// coletor público: loopback, redes privadas, link-local, CGNAT, multicast,
// documentação e faixas reservadas.
package netguard

import "net"

// IsPublicIP informa se o IP pode ser usado como destino de coleta externa.
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		if !isPublicV4(v4) {
			return false
		}
		return true
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// 2001:db8::/32 é reservado para documentação.
	if len(ip) == net.IPv6len && ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
		return false
	}
	return true
}

func isPublicV4(ip net.IP) bool {
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	switch {
	case ip[0] == 0: // 0.0.0.0/8
		return false
	case ip[0] == 10: // 10.0.0.0/8
		return false
	case ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127: // 100.64.0.0/10 (CGNAT)
		return false
	case ip[0] == 127: // loopback
		return false
	case ip[0] == 169 && ip[1] == 254: // link-local / metadata
		return false
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31: // 172.16.0.0/12
		return false
	case ip[0] == 192 && ip[1] == 168: // 192.168.0.0/16
		return false
	case ip[0] == 192 && ip[1] == 0 && ip[2] == 0: // 192.0.0.0/24
		return false
	case ip[0] == 192 && ip[1] == 0 && ip[2] == 2: // TEST-NET-1
		return false
	case ip[0] == 192 && ip[1] == 88 && ip[2] == 99: // 6to4 relay anycast
		return false
	case ip[0] == 198 && (ip[1] == 18 || ip[1] == 19): // benchmark
		return false
	case ip[0] == 198 && ip[1] == 51 && ip[2] == 100: // TEST-NET-2
		return false
	case ip[0] == 203 && ip[1] == 0 && ip[2] == 113: // TEST-NET-3
		return false
	case ip[0] >= 224: // multicast, reservado e broadcast
		return false
	}
	return true
}
