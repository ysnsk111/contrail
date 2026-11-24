package server

import (
	"net/http"
	"strings"
)

// handleAdminUserRouting dispatches to either status update or user deletion.
func (s *Server) handleAdminUserRouting(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.handleAdminDeleteUser(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/status") {
		s.handleAdminUserStatus(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
