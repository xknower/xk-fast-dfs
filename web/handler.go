package web

import (
	"net/http"
)

//
func (hs *HttpServer) Home(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	return
}
