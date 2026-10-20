package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// root meniru situs yang tidak ada: semua akses ke "/" dibalas 404 polos.
// Panel hanya bisa dibuka lewat /admin (yang mengalihkan ke /admin/login).
func (server *Server) root(response http.ResponseWriter, request *http.Request) {
	http.NotFound(response, request)
}

func formatIDR(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	digits := strconv.FormatInt(amount, 10)
	var builder strings.Builder
	length := len(digits)
	for index, digit := range digits {
		if index > 0 && (length-index)%3 == 0 {
			builder.WriteByte('.')
		}
		builder.WriteRune(digit)
	}
	if negative {
		return "-" + builder.String()
	}
	return builder.String()
}
