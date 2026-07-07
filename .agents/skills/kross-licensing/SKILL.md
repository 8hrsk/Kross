---
name: kross-licensing
description: Guides AI agents in integrating, configuring, and deploying the Kross software licensing system, including client library setup, server deployment, CLI operations, and obfuscation.
---

# Kross Licensing Integration & Operation Skill

Use this skill when you need to:
1. Integrate the Kross licensing library into a client Go application.
2. Troubleshoot or configure the Kross activation server.
3. Manage, generate, or verify license keys using the Kross CLI.
4. Set up client-side build obfuscation with `garble`.

---

## 🛠️ Client Integration Guide

To verify licenses in a client-side Go application:

### 1. Import Packages
Ensure the following packages are imported:
```go
import (
    "github.com/8hrsk/kross/pkg/client"
    "github.com/8hrsk/kross/gui"
)
```

### 2. Client Initialization
Define the embedded developer public key and initialize the client:
```go
var embeddedPublicKey = []byte{/* 32 bytes of the Ed25519 public key */}

c, err := client.New(client.Config{
    PublicKey: embeddedPublicKey,
    ServerURL: "http://localhost:8080", // Server endpoint (or empty for offline-only mode)
    AppName:   "YourAppName",            // Directory folder name for storage
})
```

### 3. Check License & Launch GUI
Run the offline check. If it fails, launch the activation window:
```go
if err := c.CheckLicense(); err != nil {
    // Show Fyne GUI activation window (blocking call)
    success := gui.ShowActivationWindow(c, "YourAppName")
    if !success {
        os.Exit(1) // User cancelled or activation failed
    }
}
```

### 4. Background Sync
Run background sync in a goroutine to sync offline activations and update the blacklist:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
c.StartBackgroundSync(ctx, logger)
```

---

## 🖥️ Server Operations

The server tracks registrations to prevent key misuse:
- **Build**: Compile using `go build ./cmd/kross-server/...`
- **Run**: `./kross-server -addr=":8080" -db="kross.db"`
- **SQLite Database**: Pure Go SQLite is initialized automatically. It uses `activations` table with unique constraint `(license_id, hwid_hash)`.
- **Validation Rules**:
  - `personal`: Allows activation on multiple HWIDs (binds to email).
  - `mass`: One-time keys. If activated on a different HWID, marks all activations for this license ID as `revoked` and blacklists the key.

---

## 🔑 CLI Usage

Use `kross-cli` to manage credentials:
- **Generate Keypair**:
  ```bash
  kross-cli keygen
  ```
- **Issue Personal License**:
  ```bash
  kross-cli issue --email="user@example.com" [--days=365]
  ```
- **Bulk Mass Licenses**:
  ```bash
  kross-cli issue --mass=100
  ```

---

## 🔒 Security & Obfuscation Best Practices

1. **HWID Storage Binding**:
   Encrypted file `license.dat` uses **AES-256-GCM** key derived from `SHA256(HWID + AppName)`. Direct file copying to other devices is not supported.
2. **Reverse-Engineering Defenses**:
   Always compile client binaries using `garble` to hide strings (like Server URLs and success messages) and randomize function structures:
   ```bash
   garble -literals -tiny -seed=random build -ldflags="-s -w" -trimpath -o dist/app main.go
   ```
