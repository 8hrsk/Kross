# Kross Server Deployment & Configuration

The Kross License Server (`cmd/kross-server`) handles online activations, prevents concurrent usage of the same license key across different hardware devices, and exposes the global revocation list (blacklist) to clients.

## Compilation

Build the server package into a binary. It compiles with a pure-Go SQLite driver, removing external GCC or CGO dependencies:
```bash
go build -o dist/kross-server ./cmd/kross-server
```

---

## Running the Server

Start the server using default settings:
```bash
./dist/kross-server
```

### Configuration Flags:
- `-addr` — Port and address the server binds to (default is `:8080`).
- `-db` — Filepath to the SQLite database (default is `kross.db`).

Example command using a custom port and pointing to `/var/lib/kross/`:
```bash
./dist/kross-server -addr="0.0.0.0:9000" -db="/var/lib/kross/licensing.db"
```

*Note: The SQLite database file and its internal schema tables/indexes are automatically initialized if they do not exist.*

---

## Database Schema (SQLite)

The server utilizes SQLite (pure-Go SQLite via the `modernc.org/sqlite` package).

### Schema Tables:
*   `activations` — Logs all successful online device activations.
    *   `license_id` (TEXT) — UUID of the license.
    *   `license_type` (TEXT) — License category (`personal` or `mass`).
    *   `email` (TEXT) — User-entered email at activation.
    *   `hwid_hash` (TEXT) — SHA-256 hash of the hardware fingerprint (HWID).
    *   `activated_at` (DATETIME) — Timestamp of registration.
    *   `revoked` (BOOLEAN) — Indicates whether this activation is blacklisted.

A unique index resides on `(license_id, hwid_hash)`. This permits the same machine to re-activate the same license key indefinitely (e.g., during application re-installs) without creating duplicate rows or triggering piracy detection.

---

## Piracy Protection Logic

When the server receives a request at `/api/v1/activate`, it applies verification checks based on the license type:

### 1. Personal License (`personal`)
- Binds to the user's email.
- Multiple active HWIDs are allowed. Users can run their software on a home PC, notebook, or office desktop simultaneously under one key.

### 2. Mass License (`mass`)
- Used for single-device codes.
- The first activation binds the license ID to the current HWID.
- Subsequent activation requests from the **same HWID** return success (re-install/activation restoration).
- Any activation request from a **different HWID** (a different computer):
  1. Automatically marks all prior activations for this license as `revoked`.
  2. Adds the license to the global blacklist.
  3. Returns a revocation payload instructing clients to deactivate the software.

---

## API Endpoints

### 1. `POST /api/v1/activate`
Invoked by clients to submit pending activations.

**Request Body (JSON):**
```json
{
  "license_id": "c1a8bc4a-9b4e-4f30-8d59-3fb9a3d46a81",
  "license_type": "mass",
  "email": "user@domain.com",
  "hwid_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

**Success Response (200 OK):**
```json
{
  "status": "activated"
}
```

**Revoked Key Response (200 OK, key has been flagged):**
```json
{
  "status": "revoked",
  "revoke_keys": ["c1a8bc4a-9b4e-4f30-8d59-3fb9a3d46a81"],
  "message": "License key revoked due to multiple hardware activations"
}
```

---

### 2. `GET /api/v1/revocations`
Polled periodically by clients to fetch the global blacklist of revoked license IDs.

**Response (200 OK):**
```json
{
  "revoked_keys": [
    "c1a8bc4a-9b4e-4f30-8d59-3fb9a3d46a81",
    "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
  ]
}
```
Client applications store these IDs locally in encrypted storage and verify their current license ID against it.
