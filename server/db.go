// Package server implements the Kross license server.
package server

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Database wraps the SQLite database for activation tracking.
type Database struct {
	db *sql.DB
}

// Activation represents a license activation record.
type Activation struct {
	ID          int64     `json:"id"`
	LicenseID   string    `json:"license_id"`
	LicenseType string    `json:"license_type"`
	Email       string    `json:"email"`
	HWIDHash    string    `json:"hwid_hash"`
	ActivatedAt time.Time `json:"activated_at"`
	Revoked     bool      `json:"revoked"`
}

// NewDatabase creates and initializes the SQLite database at the given path.
// It creates the activations table and required indices if they do not exist.
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS activations (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			license_id   TEXT NOT NULL,
			license_type TEXT NOT NULL CHECK(license_type IN ('personal', 'mass')),
			email        TEXT DEFAULT '',
			hwid_hash    TEXT NOT NULL,
			activated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			revoked      BOOLEAN DEFAULT FALSE,
			UNIQUE(license_id, hwid_hash)
		);
		CREATE INDEX IF NOT EXISTS idx_license_id ON activations(license_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Database{db: db}, nil
}

// RecordActivation records a new activation.
//
// For "mass" type: if a different HWID already exists for this license_id,
// all activations for the license are revoked and the license_id is returned
// in revokeIDs.
//
// For "personal" type: multiple HWIDs are allowed.
func (d *Database) RecordActivation(licenseID, licenseType, email, hwidHash string) (revokeIDs []string, err error) {
	// Step 1: Try INSERT with ON CONFLICT DO NOTHING.
	result, err := d.db.Exec(
		`INSERT INTO activations (license_id, license_type, email, hwid_hash)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(license_id, hwid_hash) DO NOTHING`,
		licenseID, licenseType, email, hwidHash,
	)
	if err != nil {
		return nil, fmt.Errorf("insert activation: %w", err)
	}

	// Step 2: Check rows affected — 0 means duplicate (same device).
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, nil
	}

	// Step 3: Inserted successfully.
	if licenseType == "personal" {
		// Personal licenses allow multiple devices.
		return nil, nil
	}

	// Mass license: check if any OTHER activation exists with a different HWID.
	var count int
	err = d.db.QueryRow(
		`SELECT COUNT(*) FROM activations
		 WHERE license_id = ? AND hwid_hash != ? AND revoked = FALSE`,
		licenseID, hwidHash,
	).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("check other activations: %w", err)
	}

	if count > 0 {
		// Revoke ALL activations for this license.
		if _, err := d.db.Exec(
			`UPDATE activations SET revoked = TRUE WHERE license_id = ?`,
			licenseID,
		); err != nil {
			return nil, fmt.Errorf("revoke activations: %w", err)
		}
		return []string{licenseID}, nil
	}

	return nil, nil
}

// GetActivations returns all activations for a given license ID.
func (d *Database) GetActivations(licenseID string) ([]Activation, error) {
	rows, err := d.db.Query(
		`SELECT id, license_id, license_type, email, hwid_hash, activated_at, revoked
		 FROM activations WHERE license_id = ?`,
		licenseID,
	)
	if err != nil {
		return nil, fmt.Errorf("query activations: %w", err)
	}
	defer rows.Close()

	var activations []Activation
	for rows.Next() {
		var a Activation
		if err := rows.Scan(&a.ID, &a.LicenseID, &a.LicenseType, &a.Email, &a.HWIDHash, &a.ActivatedAt, &a.Revoked); err != nil {
			return nil, fmt.Errorf("scan activation: %w", err)
		}
		activations = append(activations, a)
	}

	return activations, rows.Err()
}

// RevokeActivation marks all activations for a license as revoked.
func (d *Database) RevokeActivation(licenseID string) error {
	_, err := d.db.Exec(
		`UPDATE activations SET revoked = TRUE WHERE license_id = ?`,
		licenseID,
	)
	if err != nil {
		return fmt.Errorf("revoke activation: %w", err)
	}
	return nil
}

// GetRevokedLicenses returns all distinct revoked license IDs.
func (d *Database) GetRevokedLicenses() ([]string, error) {
	rows, err := d.db.Query(
		`SELECT DISTINCT license_id FROM activations WHERE revoked = TRUE`,
	)
	if err != nil {
		return nil, fmt.Errorf("query revoked: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan revoked: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}
