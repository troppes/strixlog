package server

import (
	"net/http"
)

type Server struct {
	port string
	mux  *http.ServeMux
}

func NewServer(port string) *Server {
	mux := http.NewServeMux()
	s := &Server{port: port, mux: mux}
	mux.HandleFunc("GET /health", handleHealth)
	return s
}

func (s *Server) Start() error {
	return http.ListenAndServe(":"+s.port, s.mux)
}
