package http

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

//routes for pkg

func (s *Server) registerPkgRoutes(r *mux.Router) {
	r.HandleFunc("/helloworld", s.handleHelloWorld).Methods("GET")
}

func (s *Server) handleHelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hello World!")
}
