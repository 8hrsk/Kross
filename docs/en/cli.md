# Kross CLI Reference

The Kross CLI tool (`cmd/kross-cli`) is used by application developers to manage cryptographic keys and issue software licenses to clients.

## Compilation

You can compile the CLI utility using the following command from the project root:
```bash
go build -o dist/kross-cli ./cmd/kross-cli
```

---

## Commands & Usage

The CLI tool supports two primary subcommands: `keygen` and `issue`.

### 1. Key Generation: `keygen`
Generates a new Ed25519 key pair used for signing licenses.

```bash
kross-cli keygen
```

**Results:**
- Generates two PEM-encoded files in the current working directory:
  - `private.key` — The private key (MUST be kept secret, used to sign licenses).
  - `public.key` — The public key (embedded into the client application to validate licenses).

---

### 2. Issue License: `issue`
Generates and signs license keys. It reads `private.key` to sign the license payload and prints the encoded key format.

#### Flags:
- `--key` (optional) — Path to the private key file (defaults to `./private.key`).
- `--email` (optional) — The customer's email address (creates a personal license).
- `--days` (optional) — Validity period of the license in days. If not provided or `<=` 0, the license is perpetual.
- `--mass` (optional) — Number of mass-activation (one-time use) keys to generate.

#### Variant A: Personal License (tied to Email)
This binds the license to a specific customer's email. The client-side application will check that the email entered by the user matches the email signed inside the license key.

```bash
kross-cli issue --email="client@example.com" --days=365
```
**Stdout output:**
```text
KROSS-MY5TA-PBP2E-FZX4A-JD7Q4-S46NX-V3DZO-U6GQM-D2H2I-C3SDA-Q===
```

#### Variant B: Perpetual Personal License
```bash
kross-cli issue --email="vip@example.com"
```

#### Variant C: Mass License Generation (Bulk Output)
Used to generate unassigned codes for third-party distributors or payment gateway automation. These keys are one-time use: when activated, they bind to the user's email and HWID on the server. Activating them on another device later will automatically revoke the key.

```bash
kross-cli issue --mass=150
```
**Results:**
- Generates a file named `licenses_YYYYMMDD_HHMMSS.txt` in the current directory.
- Writes 150 unique license keys (one key per line).
- Prints the path to the text file and total generated count.
