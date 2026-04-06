package service

import "strings"

func IsValidEthAddress(addr string) bool {
	addr = strings.TrimSpace(addr)
	if len(addr) != 42 || !strings.HasPrefix(strings.ToLower(addr), "0x") {
		return false
	}
	for _, c := range addr[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func ShortAddress(addr string) string {
	if len(addr) <= 10 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}
