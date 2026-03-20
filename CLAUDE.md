# Dead Drop - Claude Instructions

## Critical: Running Go and Other Tools

**ALWAYS use `mise exec --` to run Go and all other dev tools.** No exceptions.

```bash
# YES - always do this
mise exec -- go build ./cmd/deaddrop
mise exec -- go test -race ./...
mise exec -- go vet ./...
mise exec -- go run ./cmd/deaddrop
mise exec -- go mod tidy
mise exec -- goreleaser release --clean
mise exec -- air
mise exec -- staticcheck ./...

# NO - never run bare commands
go build ./cmd/deaddrop        # WRONG
go test ./...                  # WRONG
goreleaser release             # WRONG
```

This applies to ALL tool invocations: `go`, `goreleaser`, `air`, `staticcheck`, and any other tool managed by mise.

## Project Overview

Dead Drop encrypts secrets and generates printable PDFs with QR codes. The decryption key is handwritten after printing — the printer never sees the plaintext.

- **Language:** Go
- **Encryption:** age (filippo.io/age) with scrypt passphrase
- **Key generation:** Diceware (EFF long wordlist, crypto/rand)
- **Spec:** See `spec.md` for full details

## Project Structure

```
cmd/deaddrop/main.go     # Cobra root command, entry point
internal/crypto/         # age encryption, diceware generation
internal/pdf/            # PDF layout, QR code generation
internal/restore/        # QR scanning from images
internal/ui/             # Lipgloss styles
internal/wordlist/       # Embedded EFF wordlist
```

## Build & Test

```bash
mise exec -- go test -v -race -cover ./...
mise exec -- go build -ldflags="-s -w -X main.version=dev" -o ./bin/deaddrop ./cmd/deaddrop
mise exec -- go vet ./...
```

Or via mise tasks:

```bash
mise run test
mise run build
mise run lint
mise run dev
```

## Conventions

- Single static binary, `CGO_ENABLED=0`
- No TUI — this is a quick CLI tool, not an interactive app
- Use Charm libraries (Lipgloss, Huh, charmbracelet/log) for pretty terminal output
- All crypto uses `crypto/rand`, never `math/rand`
- age format means users can decrypt without Dead Drop using just the `age` CLI
