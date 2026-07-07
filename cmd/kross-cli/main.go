package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/user/kross/internal/crypto"
	"github.com/user/kross/internal/license"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "keygen":
		handleKeygen()
	case "issue":
		handleIssue()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Kross License Manager CLI

Usage:
  kross-cli keygen
  kross-cli issue --email=<email> [--days=<days>] [--key=<private.key>]
  kross-cli issue --mass=<count> [--key=<private.key>]

Commands:
  keygen    Generate a new Ed25519 key pair
  issue     Issue license(s)

Flags:
  --email   Email address for personal license
  --days    License validity in days (0 or omit = perpetual)
  --mass    Number of one-time-use licenses to generate
  --key     Path to private key file (default: private.key)
`)
}

func handleKeygen() {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating keys: %v\n", err)
		os.Exit(1)
	}

	if err := crypto.SavePrivateKey("private.key", priv); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving private key: %v\n", err)
		os.Exit(1)
	}
	if err := crypto.SavePublicKey("public.key", pub); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving public key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Keys generated successfully:")
	fmt.Println(" - private.key (Keep this secret!)")
	fmt.Println(" - public.key (Embed this in your app)")
}

func handleIssue() {
	issueCmd := flag.NewFlagSet("issue", flag.ExitOnError)
	email := issueCmd.String("email", "", "Email address for personal license")
	days := issueCmd.Int("days", 0, "License validity in days (0 = perpetual)")
	mass := issueCmd.Int("mass", 0, "Number of one-time-use mass licenses to generate")
	keyPath := issueCmd.String("key", "private.key", "Path to private key file")

	// Parse arguments after the subcommand
	issueCmd.Parse(os.Args[2:])

	if *mass == 0 && *email == "" {
		fmt.Fprintln(os.Stderr, "Error: Must specify either --email or --mass")
		issueCmd.Usage()
		os.Exit(1)
	}

	priv, err := crypto.LoadPrivateKey(*keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading private key from %s: %v\n", *keyPath, err)
		os.Exit(1)
	}

	if *mass > 0 {
		generateMassLicenses(*mass, priv)
	} else {
		generatePersonalLicense(*email, *days, priv)
	}
}

func generatePersonalLicense(email string, days int, priv ed25519.PrivateKey) {
	lic := license.NewPersonalLicense(email, days)
	signedLic, err := license.SignLicense(lic, priv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing license: %v\n", err)
		os.Exit(1)
	}

	encoded, err := license.Encode(signedLic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding license: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated Personal License Key:")
	fmt.Println(encoded)
}

func generateMassLicenses(count int, priv ed25519.PrivateKey) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("licenses_%s.txt", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	for i := 0; i < count; i++ {
		lic := license.NewMassLicense()
		signedLic, err := license.SignLicense(lic, priv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error signing mass license %d: %v\n", i+1, err)
			continue
		}

		encoded, err := license.Encode(signedLic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding mass license %d: %v\n", i+1, err)
			continue
		}

		fmt.Fprintln(file, encoded)
	}

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("Generated %d mass licenses.\n", count)
	fmt.Printf("Saved to: %s\n", absPath)
}
