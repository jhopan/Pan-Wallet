package httpapi

import (
	"encoding/json"
	"net/http"
)

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	return &Server{mux: mux}
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.mux.ServeHTTP(response, request)
}

func health(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(response).Encode(map[string]string{"status": "ok"})
}
