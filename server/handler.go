package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Server is the HTTP server for the Kross license API.
type Server struct {
	db     *Database
	mux    *http.ServeMux
	logger *slog.Logger
}

// ActivateRequest is the JSON body for POST /api/v1/activate.
type ActivateRequest struct {
	LicenseID   string `json:"license_id"`
	LicenseType string `json:"license_type"`
	Email       string `json:"email"`
	HWIDHash    string `json:"hwid_hash"`
}

// ActivateResponse is the JSON response for POST /api/v1/activate.
type ActivateResponse struct {
	Status     string   `json:"status"`
	RevokeKeys []string `json:"revoke_keys,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// RevocationsResponse is the JSON response for GET /api/v1/revocations.
type RevocationsResponse struct {
	RevokedKeys []string `json:"revoked_keys"`
}

// NewServer creates a new Server with the given database and logger.
func NewServer(db *Database, logger *slog.Logger) *Server {
	s := &Server{
		db:     db,
		mux:    http.NewServeMux(),
		logger: logger,
	}
	s.mux.HandleFunc("POST /api/v1/activate", s.handleActivate)
	s.mux.HandleFunc("GET /api/v1/revocations", s.handleRevocations)
	return s
}

// ServeHTTP delegates to the internal mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleActivate processes activation requests.
func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	var req ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, ActivateResponse{
			Status:  "error",
			Message: "invalid JSON body",
		})
		return
	}

	// Validate required fields.
	if req.LicenseID == "" || req.LicenseType == "" || req.HWIDHash == "" {
		s.writeJSON(w, http.StatusBadRequest, ActivateResponse{
			Status:  "error",
			Message: "license_id, license_type, and hwid_hash are required",
		})
		return
	}

	// Validate license_type.
	if req.LicenseType != "personal" && req.LicenseType != "mass" {
		s.writeJSON(w, http.StatusBadRequest, ActivateResponse{
			Status:  "error",
			Message: "license_type must be 'personal' or 'mass'",
		})
		return
	}

	revokeIDs, err := s.db.RecordActivation(req.LicenseID, req.LicenseType, req.Email, req.HWIDHash)
	if err != nil {
		s.logger.Error("record activation failed", "error", err)
		s.writeJSON(w, http.StatusInternalServerError, ActivateResponse{
			Status:  "error",
			Message: "internal server error",
		})
		return
	}

	resp := ActivateResponse{Status: "ok"}
	if len(revokeIDs) > 0 {
		resp.RevokeKeys = revokeIDs
		resp.Message = "previous activations revoked"
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleRevocations returns all revoked license IDs.
func (s *Server) handleRevocations(w http.ResponseWriter, _ *http.Request) {
	ids, err := s.db.GetRevokedLicenses()
	if err != nil {
		s.logger.Error("get revoked licenses failed", "error", err)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
		return
	}

	if ids == nil {
		ids = []string{}
	}

	s.writeJSON(w, http.StatusOK, RevocationsResponse{RevokedKeys: ids})
}

// writeJSON encodes v as JSON and writes it to w with the given status code.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("write JSON response failed", "error", err)
	}
}
