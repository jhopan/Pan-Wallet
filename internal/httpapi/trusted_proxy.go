package httpapi

import (
	"net"
	"strings"
)

// trustedProxies berisi jaringan reverse proxy yang boleh dipercaya
// mengirim header X-Forwarded-For / CF-Connecting-IP.
//
// Tanpa daftar ini, siapa pun bisa memalsukan IP lewat header dan
// melewati rate limit sepenuhnya (sudah terbukti: 100/100 request lolos).
type trustedProxies struct {
	networks []*net.IPNet
}

// newTrustedProxies membuat daftar dari string CIDR atau IP tunggal.
// Entri kosong diabaikan.
func newTrustedProxies(entries []string) *trustedProxies {
	trusted := &trustedProxies{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			// IP tunggal: jadikan /32 atau /128.
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			trusted.networks = append(trusted.networks, network)
		}
	}
	return trusted
}

// contains melaporkan apakah IP termasuk proxy tepercaya.
func (trusted *trustedProxies) contains(ip string) bool {
	if trusted == nil || len(trusted.networks) == 0 {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	for _, network := range trusted.networks {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

// empty melaporkan apakah tidak ada proxy tepercaya yang dikonfigurasi.
func (trusted *trustedProxies) empty() bool {
	return trusted == nil || len(trusted.networks) == 0
}
