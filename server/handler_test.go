package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestServer creates a Server backed by a temporary SQLite database.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(db, logger)
}

// postActivate is a helper that sends a POST /api/v1/activate request.
func postActivate(t *testing.T, srv *Server, req ActivateRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/activate", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	return w
}

func TestActivatePersonalLicense(t *testing.T) {
	srv := newTestServer(t)

	w := postActivate(t, srv, ActivateRequest{
		LicenseID:   "lic-personal-001",
		LicenseType: "personal",
		Email:       "user@example.com",
		HWIDHash:    "hwid-aaa",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ActivateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if len(resp.RevokeKeys) != 0 {
		t.Errorf("expected no revoke_keys, got %v", resp.RevokeKeys)
	}
}

func TestActivateDuplicateDevice(t *testing.T) {
	srv := newTestServer(t)

	req := ActivateRequest{
		LicenseID:   "lic-dup-001",
		LicenseType: "personal",
		Email:       "user@example.com",
		HWIDHash:    "hwid-bbb",
	}

	// First activation.
	w := postActivate(t, srv, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first activate: expected 200, got %d", w.Code)
	}

	// Duplicate activation (same device).
	w = postActivate(t, srv, req)
	if w.Code != http.StatusOK {
		t.Fatalf("duplicate activate: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ActivateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if len(resp.RevokeKeys) != 0 {
		t.Errorf("expected no revoke_keys for duplicate, got %v", resp.RevokeKeys)
	}
}

func TestActivateMassKeyDifferentDevice(t *testing.T) {
	srv := newTestServer(t)

	// First activation on device A.
	w := postActivate(t, srv, ActivateRequest{
		LicenseID:   "lic-mass-001",
		LicenseType: "mass",
		Email:       "user@example.com",
		HWIDHash:    "hwid-device-a",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("first activate: expected 200, got %d", w.Code)
	}

	// Second activation on device B → should trigger revocation.
	w = postActivate(t, srv, ActivateRequest{
		LicenseID:   "lic-mass-001",
		LicenseType: "mass",
		Email:       "user@example.com",
		HWIDHash:    "hwid-device-b",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("second activate: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ActivateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if len(resp.RevokeKeys) != 1 || resp.RevokeKeys[0] != "lic-mass-001" {
		t.Errorf("expected revoke_keys=[lic-mass-001], got %v", resp.RevokeKeys)
	}
}

func TestActivateMissingFields(t *testing.T) {
	srv := newTestServer(t)

	cases := []struct {
		name string
		req  ActivateRequest
	}{
		{
			name: "missing license_id",
			req:  ActivateRequest{LicenseType: "personal", HWIDHash: "hwid"},
		},
		{
			name: "missing license_type",
			req:  ActivateRequest{LicenseID: "lic-001", HWIDHash: "hwid"},
		},
		{
			name: "missing hwid_hash",
			req:  ActivateRequest{LicenseID: "lic-001", LicenseType: "personal"},
		},
		{
			name: "invalid license_type",
			req:  ActivateRequest{LicenseID: "lic-001", LicenseType: "enterprise", HWIDHash: "hwid"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postActivate(t, srv, tc.req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRevocations(t *testing.T) {
	srv := newTestServer(t)

	// Activate mass license on two devices to trigger revocation.
	postActivate(t, srv, ActivateRequest{
		LicenseID:   "lic-rev-001",
		LicenseType: "mass",
		Email:       "user@example.com",
		HWIDHash:    "hwid-x",
	})
	postActivate(t, srv, ActivateRequest{
		LicenseID:   "lic-rev-001",
		LicenseType: "mass",
		Email:       "user@example.com",
		HWIDHash:    "hwid-y",
	})

	// GET /api/v1/revocations
	r := httptest.NewRequest(http.MethodGet, "/api/v1/revocations", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp RevocationsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.RevokedKeys) != 1 || resp.RevokedKeys[0] != "lic-rev-001" {
		t.Errorf("expected revoked_keys=[lic-rev-001], got %v", resp.RevokedKeys)
	}
}
