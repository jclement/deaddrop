# Dead Drop

**Print your secrets without trusting your printer.**

Dead Drop is a single-binary CLI tool that encrypts sensitive data and generates a printable PDF containing a QR code of the ciphertext -- with a blank space for the decryption key to be handwritten. The key never touches the printer. The secret never leaves the page unencrypted.

---

## The Problem

You have a secret -- a seed phrase, a recovery key, a password, an API token -- and you want a physical backup. But:

- Printing it directly means your printer (and its memory, its network, its cloud service) sees the plaintext
- Writing it by hand is error-prone and tedious for long secrets
- Existing tools either require trust in too many systems or produce output that's impossible to restore without the exact same software

## The Solution

Dead Drop encrypts your secret with a strong diceware passphrase, encodes the ciphertext as a QR code, and generates a clean PDF letter with:

1. **The QR code** -- containing the encrypted payload
2. **A Z85-encoded text block** -- a human-typeable fallback if the QR code is ever unscannable
3. **A blank field** -- where you handwrite the diceware key after printing
4. **Restore instructions** -- enough detail to decrypt without Dead Drop itself, using only standard tools

The printer sees only ciphertext. The key exists only in your head and on the paper, written by your hand.

---

## Usage

### Encrypt (create a dead drop)

```bash
# From a file
deaddrop create secret.txt

# From stdin
cat secret.txt | deaddrop create -

# Interactive prompt (no file, no pipe -- just type/paste)
deaddrop create
```

**Output:** A PDF file (default: `deaddrop-<timestamp>.pdf`) and the diceware key displayed on screen.

### Verify & Decrypt (restore from a dead drop)

```bash
# Scan QR code from a photo/screenshot of the printed letter and decrypt
deaddrop restore photo-of-letter.jpg
deaddrop restore scan.png

# Paste the Z85-encoded text block manually (interactive)
deaddrop restore

# Pipe in raw age ciphertext
cat secret.age | deaddrop restore -
```

The `restore` command:
1. Accepts an image file (PNG, JPEG, PDF) -- scans for QR code automatically
2. If no image given, prompts to paste the Z85-encoded text block from the PDF
3. Prompts for the passphrase (the one handwritten on the paper)
4. Decrypts and outputs the original secret

This doubles as a **verification** tool -- after printing, snap a photo of the paper and run `deaddrop restore photo.jpg` to confirm the QR code is scannable and the data round-trips correctly.

### Other commands

```bash
deaddrop version          # Full version, commit, Go version, OS/arch
deaddrop update           # Self-update from GitHub Releases
deaddrop --help           # Help
```

---

## Cryptography

### Primitives

| Component | Choice | Rationale |
|-----------|--------|-----------|
| **Encryption** | [age](https://age-encryption.org/) (scrypt recipient) | Modern, audited, simple. The `age` CLI can decrypt our output -- no vendor lock-in. |
| **Key derivation** | scrypt (via age's passphrase mode) | Memory-hard KDF, resistant to GPU/ASIC brute-force |
| **Passphrase** | [Diceware](https://theworld.com/~reinhold/diceware.html) (6+ words, EFF long wordlist) | High entropy (~77 bits for 6 words), human-readable, hand-writable |
| **QR encoding** | Binary mode QR | Maximum data density for the encrypted payload |

### Implementation

Use [`filippo.io/age`](https://pkg.go.dev/filippo.io/age) -- the reference Go implementation:

```go
import "filippo.io/age"

// Encrypt
recipient, err := age.NewScryptRecipient(passphrase)
recipient.SetWorkFactor(18) // ~1 second on modern hardware
w, err := age.Encrypt(out, recipient)

// Decrypt
identity, err := age.NewScryptIdentity(passphrase)
r, err := age.Decrypt(in, identity)
```

### Diceware Generation

Use the EFF long wordlist (7776 words, 12.9 bits per word). Generate 6 words minimum (~77 bits of entropy). Use `crypto/rand` exclusively -- never `math/rand`.

```go
import "crypto/rand"

func generateDiceware(wordCount int) (string, error) {
    words := make([]string, wordCount)
    for i := range words {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(effWordlist))))
        if err != nil {
            return "", err
        }
        words[i] = effWordlist[n.Int64()]
    }
    return strings.Join(words, "-"), nil
}
```

---

## PDF Output

The PDF should look **clean, professional, and trustworthy** -- like a document you'd keep in a safe. Think: minimalist design, generous whitespace, crisp typography. This is a security document; it should feel serious.

### Design Principles

- **Monochrome.** Black, white, and grays only. No color -- this will be printed, possibly photocopied, and color adds nothing.
- **Generous margins.** At least 1 inch / 25mm on all sides. The document needs to breathe.
- **Clear hierarchy.** The QR code is the hero. The passphrase field is prominent. Metadata and instructions are secondary.
- **Embedded fonts.** Use a clean sans-serif (e.g., embed Inter or use Helvetica) for headings and a monospace font (e.g., embed JetBrains Mono or use Courier) for the encoded data block. Fonts must be embedded in the PDF for consistent rendering.
- **No branding clutter.** A subtle "Generated by Dead Drop" footer at most. The document is about the secret, not the tool.

### Layout

```
+--------------------------------------------------------------+
|                                                              |
|   DEAD DROP                                    2026-03-20    |
|   -------------------------------------------------------    |
|                                                              |
|   Label: seed-phrase-backup                                  |
|   Algorithm: age (scrypt, work factor 18)                    |
|                                                              |
|                   +-------------------+                      |
|                   |                   |                      |
|                   |                   |                      |
|                   |     QR CODE       |                      |
|                   |    (encrypted     |                      |
|                   |     payload)      |                      |
|                   |                   |                      |
|                   |                   |                      |
|                   +-------------------+                      |
|                                                              |
|   +------------------------------------------------------+   |
|   |                                                      |   |
|   |  PASSPHRASE                                          |   |
|   |                                                      |   |
|   |  ________________________________________________    |   |
|   |                                                      |   |
|   |  ________________________________________________    |   |
|   |                                                      |   |
|   |  Write the passphrase here by hand.                  |   |
|   |  Do NOT type or print it.                            |   |
|   |                                                      |   |
|   +------------------------------------------------------+   |
|                                                              |
|   -- Encoded Payload (fallback) -------------------------    |
|                                                              |
|   If the QR code is damaged or unscannable, manually         |
|   type this Z85-encoded text into a file and decode:         |
|                                                              |
|   +------------------------------------------------------+   |
|   |  rA&H9]o3vB%kW8mP#xQ2nL7jF!dY5tC0gU4sE6wR1b        |   |
|   |  N9iM3aK8pJ2cX7hV5fT0zDqL6yW4eO1nS9mB3gR8kP        |   |
|   |  2jH7xC5vA0tFwQ9eL3nY8bK6dM1sR4pJ7gU0hX2cV         |   |
|   |  ...                                                 |   |
|   +------------------------------------------------------+   |
|                                                              |
|   -- Restore Without Dead Drop --------------------------    |
|                                                              |
|   1. Scan the QR code (or type the Z85 block above)         |
|   2. If using the Z85 text, decode it to binary first:       |
|      deaddrop restore  (or any Z85/Base85 decoder)           |
|   3. Save the binary output to a file (e.g. secret.age)     |
|   4. Run: age -d secret.age                                 |
|   5. Enter the passphrase written above                      |
|                                                              |
|   The payload is age-encrypted (age-encryption.org)          |
|   using a passphrase/scrypt recipient. Any age-compatible    |
|   tool can decrypt it.                                       |
|                                                              |
|                              Generated by Dead Drop          |
|                                                              |
+--------------------------------------------------------------+
```

### Fallback Encoding: Z85

QR codes can be damaged, fade over time, or fail to scan from photocopies. The PDF includes the full encrypted payload as a **Z85-encoded text block** -- a human-readable, typeable fallback.

**Why Z85 (ZeroMQ Base85)?**

| Encoding | Efficiency | Ambiguous chars? | Notes |
|----------|-----------|-----------------|-------|
| Hex | 50% | No | Too verbose -- doubles payload size |
| Base64 | 75% | Yes (0/O, l/1/I) | Common but error-prone when retyping |
| Ascii85 | 80% | Some | Contains whitespace-adjacent chars |
| **Z85** | **80%** | **No** | Clean charset, no quotes/backslash/ambiguous chars |

Z85 uses 85 carefully chosen printable ASCII characters: `0-9`, `a-z`, `A-Z`, and `.-:+=^!/*?&<>()[]{}@%$#`. No quotes, no backslash, no space, no tilde -- all characters that survive photocopying, OCR, and manual typing.

**Formatted for readability:** The Z85 block on the PDF is printed in a monospace font, broken into fixed-width lines (50 chars per line), with a light gray alternating-row background to help track lines visually when retyping.

```go
import "github.com/tilinna/z85"

// Encode: pad ciphertext to multiple of 4 bytes, then Z85 encode
func encodeZ85(data []byte) (string, error) {
    // Z85 requires input length divisible by 4
    padded := padTo4(data)
    dst := make([]byte, z85.EncodedLen(len(padded)))
    _, err := z85.Encode(dst, padded)
    return string(dst), err
}
```

### PDF Generation

Use [`go-pdf/fpdf`](https://github.com/go-pdf/fpdf) (maintained fork of jung-kurt/gofpdf) for PDF generation and [`skip2/go-qrcode`](https://github.com/skip2/go-qrcode) for QR codes. Both are pure Go -- no CGO needed.

For embedded fonts, use fpdf's `AddUTF8Font` with font files compiled into the binary via `go:embed`.

### Payload Size Limits

QR codes have capacity limits. For a binary-mode QR code at error correction level L:

| Version | Max bytes |
|---------|-----------|
| 40 (largest) | 2,953 bytes |

Age-encrypted output adds ~200 bytes of overhead. So the practical limit for input data is ~2,700 bytes -- plenty for seed phrases, keys, passwords, and short documents. If the payload exceeds QR capacity, Dead Drop should:

1. Warn the user
2. Offer to split across multiple QR codes (multi-page PDF)
3. Each page is self-contained with its own QR code, Z85 block, restore instructions, and page number (e.g., "Page 1 of 3")
4. The passphrase field only appears on page 1 (it's the same key for all pages)

---

## CLI Design

### Cobra Command Structure

```
deaddrop
|-- create    # Encrypt and generate PDF (default command)
|-- restore   # Decrypt from QR image, Z85 text, or raw ciphertext
|-- version   # Detailed version info
+-- update    # Self-update from GitHub Releases
```

### Flags

```
Global:
  -d, --debug           Enable debug logging
  -q, --quiet           Suppress non-essential output
      --config string   Config file path
      --json            Output as JSON (for scriptability)

create:
  -o, --output string   Output PDF path (default: deaddrop-<timestamp>.pdf)
  -l, --label string    Label for the document (default: filename or "secret")
  -w, --words int       Number of diceware words (default: 6, min: 5)
      --work-factor int Age scrypt work factor (default: 18)
      --no-instructions Omit restore instructions from PDF

restore:
  (no special flags -- accepts optional image path as argument)
```

### Terminal UX

Use [Charm](https://charm.sh/) libraries for a polished CLI experience -- but **no full TUI**. Dead Drop is a quick-in, quick-out tool. Interactive prompts via [Huh](https://github.com/charmbracelet/huh), styled output via [Lipgloss](https://github.com/charmbracelet/lipgloss), and [charmbracelet/log](https://github.com/charmbracelet/log) for pretty logging.

```go
// Styled output after encryption
fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7AA2F7")).Render("Done. Dead drop created."))
fmt.Printf("  PDF:        %s\n", outputPath)
fmt.Printf("  Label:      %s\n", label)
fmt.Println()
fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5C07B")).Render("Write down this passphrase and destroy this terminal output:"))
fmt.Println()
fmt.Printf("  %s\n", passphrase)
fmt.Println()
```

### Piping & Scriptability

When stdin isn't a TTY, skip interactive prompts and behave as a pure CLI:

```go
import "golang.org/x/term"

if !term.IsTerminal(int(os.Stdin.Fd())) {
    // Read from stdin, output passphrase to stdout, PDF to file
    // No interactive prompts, no color
}
```

Support `--json` for machine-readable output:

```json
{
  "pdf": "deaddrop-20260320-143022.pdf",
  "label": "seed-phrase",
  "passphrase": "correct-horse-battery-staple-lunar-orbit",
  "words": 6,
  "algorithm": "age-scrypt",
  "payload_bytes": 1247
}
```

---

## Stack

| Component | Choice | Notes |
|-----------|--------|-------|
| **CLI framework** | [Cobra](https://github.com/spf13/cobra) | Commands, flags, completions, help generation |
| **Styled output** | [Lipgloss](https://github.com/charmbracelet/lipgloss) | Colors, borders, formatting |
| **Interactive prompts** | [Huh](https://github.com/charmbracelet/huh) | For secret input, confirmations |
| **Pretty logging** | [charmbracelet/log](https://github.com/charmbracelet/log) | Colorful, structured stderr logging |
| **Encryption** | [filippo.io/age](https://pkg.go.dev/filippo.io/age) | Reference age implementation |
| **PDF generation** | [go-pdf/fpdf](https://github.com/go-pdf/fpdf) | Pure Go PDF creation |
| **QR codes** | [skip2/go-qrcode](https://github.com/skip2/go-qrcode) | Pure Go QR generation |
| **QR scanning** | [makiuchi-d/gozxing](https://github.com/makiuchi-d/gozxing) | Pure Go ZXing port for restore from images |
| **Z85 encoding** | [tilinna/z85](https://github.com/tilinna/z85) | ZeroMQ Base85 encode/decode |
| **Config** | [Viper](https://github.com/spf13/viper) | YAML config + env vars |
| **Self-update** | [creativeprojects/go-selfupdate](https://github.com/creativeprojects/go-selfupdate) | GitHub Releases auto-update |
| **Releases** | [GoReleaser](https://goreleaser.com/) | Multi-platform binaries + Homebrew tap |

---

## Project Structure

```
.
|-- cmd/
|   +-- deaddrop/
|       +-- main.go              # Cobra root command, config loading
|-- internal/
|   |-- crypto/
|   |   |-- encrypt.go           # age encryption/decryption
|   |   +-- diceware.go          # Diceware passphrase generation (EFF wordlist)
|   |-- pdf/
|   |   |-- generate.go          # PDF layout and generation
|   |   +-- qr.go                # QR code encoding
|   |-- restore/
|   |   |-- scan.go              # QR code scanning from images
|   |   +-- z85.go               # Z85 encode/decode for fallback text
|   |-- ui/
|   |   +-- styles.go            # Lipgloss style definitions
|   +-- wordlist/
|       +-- eff.go               # Embedded EFF long wordlist
|-- mise.toml
|-- .goreleaser.yaml
|-- .github/
|   +-- workflows/
|       |-- ci.yml
|       +-- release.yml
|-- Dockerfile
|-- CLAUDE.md
+-- README.md
```

---

## Build

### Single static binary. No CGO.

```bash
CGO_ENABLED=0 go build -ldflags="-s -w \
  -X main.version=v1.0.0 \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  ./cmd/deaddrop
```

### Version embedding

```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

Display:
- `deaddrop --version` -> `deaddrop v1.0.0 (abc1234, 2026-03-20T10:30:00Z)`
- `deaddrop version` -> full details including Go version and OS/arch

---

## Tooling & Development

### mise.toml

```toml
[tools]
go = "latest"
goreleaser = "latest"
"go:github.com/air-verse/air" = "latest"

[tasks.dev]
description = "Run with live reload"
run = "air"

[tasks.test]
description = "Run tests with race detector"
run = "go test -v -race -cover ./..."

[tasks.lint]
description = "Run linters"
run = "go vet ./... && go run honnef.co/go/tools/cmd/staticcheck@latest ./..."

[tasks.build]
description = "Build binary"
run = 'go build -ldflags="-s -w -X main.version=dev" -o ./bin/deaddrop ./cmd/deaddrop'

[tasks.release]
description = "Tag and push a release"
run = """
#!/bin/bash
set -e
current=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Current: $current"
major=$(echo $current | sed 's/v//' | cut -d. -f1)
minor=$(echo $current | cut -d. -f2)
patch=$(echo $current | cut -d. -f3)
echo "  1) patch -> v${major}.${minor}.$((patch+1))"
echo "  2) minor -> v${major}.$((minor+1)).0"
echo "  3) major -> v$((major+1)).0.0"
read -p "Choice [1]: " c; c=${c:-1}
case $c in
  1) v="v${major}.${minor}.$((patch+1))" ;;
  2) v="v${major}.$((minor+1)).0" ;;
  3) v="v$((major+1)).0.0" ;;
  *) echo "Invalid"; exit 1 ;;
esac
git tag -a "$v" -m "Release $v"
read -p "Push $v? [Y/n]: " p; p=${p:-Y}
[[ $p =~ ^[Yy]$ ]] && git push origin "$v"
"""
```

### .air.toml

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = 'go build -ldflags="-X main.version=dev" -o ./tmp/deaddrop ./cmd/deaddrop'
bin = "./tmp/deaddrop"
include_ext = ["go"]
include_dir = ["cmd", "internal"]
exclude_dir = ["tmp", "vendor", ".git", "dist", "bin"]
delay = 500
stop_on_error = true
send_interrupt = true
kill_delay = 500
```

---

## CI/CD

### .github/workflows/ci.yml

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v2
        with: { experimental: true }
      - run: go mod download
      - run: mise run lint
      - run: mise run test
      - run: mise run build
```

### .github/workflows/release.yml

```yaml
name: Release
on:
  push:
    tags: ["v*"]

permissions:
  contents: write
  packages: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: jdx/mise-action@v2
        with: { experimental: true }
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## .goreleaser.yaml

```yaml
version: 2
project_name: deaddrop

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/deaddrop
    binary: deaddrop
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}} -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

brews:
  - repository:
      owner: jclement
      name: homebrew-tap
    homepage: "https://github.com/jclement/deaddrop"
    description: "Print your secrets without trusting your printer"
    install: |
      bin.install "deaddrop"

dockers:
  - goos: linux
    goarch: amd64
    dockerfile: Dockerfile.goreleaser
    use: buildx
    build_flag_templates: ["--platform=linux/amd64"]
    image_templates:
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:{{ .Version }}-amd64"
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:latest-amd64"
  - goos: linux
    goarch: arm64
    dockerfile: Dockerfile.goreleaser
    use: buildx
    build_flag_templates: ["--platform=linux/arm64"]
    image_templates:
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:{{ .Version }}-arm64"
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:latest-arm64"

docker_manifests:
  - name_template: "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:{{ .Version }}"
    image_templates:
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:{{ .Version }}-amd64"
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:{{ .Version }}-arm64"
  - name_template: "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:latest"
    image_templates:
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:latest-amd64"
      - "ghcr.io/{{ .Env.GITHUB_REPOSITORY_OWNER }}/deaddrop:latest-arm64"

changelog:
  sort: asc
  filters:
    exclude: ["^docs:", "^test:", "^ci:", "^chore:"]

release:
  prerelease: auto
```

---

## Docker

### Dockerfile

```dockerfile
FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o deaddrop ./cmd/deaddrop

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/deaddrop /usr/local/bin/deaddrop
ENTRYPOINT ["deaddrop"]
```

### Dockerfile.goreleaser

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY deaddrop /usr/local/bin/deaddrop
ENTRYPOINT ["deaddrop"]
```

### Usage

```bash
# Quick try
docker run -it --rm ghcr.io/jclement/deaddrop create

# Mount files
docker run -it --rm -v "$(pwd):/work" -w /work ghcr.io/jclement/deaddrop create secret.txt
```

---

## Self-Update

Background update check on every run (non-blocking). Use [`creativeprojects/go-selfupdate`](https://github.com/creativeprojects/go-selfupdate):

```go
// Check at startup, print if newer version exists
source, _ := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
updater, _ := selfupdate.NewUpdater(selfupdate.Config{Source: source})
latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug("jclement/deaddrop"))
if err == nil && found && latest.GreaterThan(version) {
    fmt.Fprintf(os.Stderr, "Update available: %s -> %s (run `deaddrop update`)\n", version, latest.Version())
}
```

The `deaddrop update` subcommand performs the actual in-place binary replacement.

---

## Testing

### Unit tests
Standard `testing` stdlib. Table-driven. Race detector always on (`go test -race`).

### Key test scenarios

| Test | What it verifies |
|------|-----------------|
| Round-trip encrypt/decrypt | Encrypt data -> decrypt with same passphrase -> original data |
| Wrong passphrase rejection | Decrypt with wrong passphrase -> error |
| Diceware word count | Generated passphrase has exactly N words |
| Diceware entropy source | Uses crypto/rand, not math/rand |
| QR encode/decode round-trip | Encode ciphertext -> QR image -> scan QR -> same ciphertext |
| Z85 encode/decode round-trip | Encode ciphertext -> Z85 text -> decode Z85 -> same ciphertext |
| QR payload overflow | Input too large for single QR -> multi-page split |
| PDF generation | Creates valid PDF file with expected elements |
| Restore from image | Create PDF -> render to image -> restore from image -> original secret |
| Stdin piping | `echo "secret" | deaddrop create -` works without TTY |
| JSON output | `--json` flag produces valid, parseable JSON |

---

## Security Considerations

- **Passphrase never written to disk.** Displayed on screen once, then the user writes it by hand.
- **Memory hygiene.** Zero passphrase bytes after use where possible (best-effort in Go's GC environment).
- **No telemetry.** No network calls except self-update check (which can be disabled via config).
- **Deterministic builds.** GoReleaser + CGO_ENABLED=0 = reproducible, auditable binaries.
- **Escape hatch.** The restore instructions on the PDF describe how to decrypt using only the `age` CLI -- no vendor lock-in.
- **Verify after printing.** The `restore` command doubles as a verification tool -- photograph the printed page and confirm the data round-trips before relying on it.

---

## Configuration

YAML config at `~/.config/deaddrop/config.yaml`. Auto-created with commented defaults on first run.

```yaml
# deaddrop configuration

# words: 6          # Number of diceware words (min: 5)
# work_factor: 18   # age scrypt work factor
# output_dir: .     # Default output directory for PDFs
# check_updates: true
# log_level: info   # debug | info | warn | error
```

Follow platform conventions:

| Purpose | Linux | macOS | Env Override |
|---------|-------|-------|-------------|
| Config | `~/.config/deaddrop/` | `~/Library/Application Support/deaddrop/` | `XDG_CONFIG_HOME` |
| State/logs | `~/.local/state/deaddrop/` | `~/Library/Application Support/deaddrop/` | `XDG_STATE_HOME` |
