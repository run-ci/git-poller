package http

import (
	"net/http"
)

// GetRoot is just a basic handler that signals to clients that
// the server is listening.
func getRoot(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	return
}
