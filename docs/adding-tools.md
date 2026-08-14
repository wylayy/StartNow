# Adding a New Tool

Tools are pure metadata — no Go logic per tool. Add one entry to
`Tools()` in `internal/catalog/catalog.go`, run `go test ./...`, done.

```go
{
    Name:        "lazygit",
    DisplayName: "LazyGit",
    Category:    "Tools",
    Description: "Terminal UI for git",
    VersionCmd:  []string{"lazygit", "--version"},
    Kind:        KindArchive,
    Source:      SourceGitHub,
    Repo:        "jesseduffield/lazygit",
    AssetTemplate: "lazygit_{{.Version}}",
    ArchiveURL:    "https://github.com/{{.Repo}}/releases/download/{{.Tag}}/{{.Asset}}",
    ChecksumAsset: []string{"checksums.txt", "SHA256SUMS"},
},
```

## Tool fields

| Field          | Purpose                                                        |
| -------------- | -------------------------------------------------------------- |
| `Name`         | unique id, used in the manifest and events                     |
| `DisplayName`  | shown in the table                                             |
| `Category`     | `Languages`, `Runtimes`, `Tools`, … (shown as a table column)  |
| `Description`  | one-liner shown in the table and details panel                 |
| `VersionCmd`   | probe command, e.g. `{"go", "version"}` — detected at startup  |
| `Kind`         | how the tool is installed (below)                              |
| `Source`       | version source for archive kind (below)                        |
| `Version`      | optional pinned version; empty = latest                        |

## Kinds

### `KindArchive` — direct download

Downloads a versioned archive from an official source, verifies its sha256,
extracts it into `~/.startnow/tools/<name>`, and symlinks the binaries into
`~/.startnow/bin`. Versioned, supports updates and uninstalls.

| Source        | What it needs                                                                      |
| ------------- | ---------------------------------------------------------------------------------- |
| `SourceGoDev` | nothing — the go.dev API supplies filename + checksum. `ArchiveURL` template only |
| `SourceNodeJS`| `AssetTemplate`, `ArchiveURL`, `ChecksumsURL` (SHASUMS256.txt), `BinRel: "bin"`   |
| `SourceGitHub`| `Repo` ("owner/repo"), `AssetTemplate`, `ArchiveURL`, optional `ChecksumAsset`    |

Template placeholders: `{{.Name}} {{.Version}} {{.Tag}} {{.Asset}} {{.OS}}
{{.Arch}} {{.Repo}} {{.Prefix}} {{.BinDir}}`.

Example (Go):

```go
{
    Name: "go", DisplayName: "Go", Category: "Languages",
    Description: "Go toolchain",
    VersionCmd:  []string{"go", "version"},
    Kind:   KindArchive,
    Source: SourceGoDev,
    ArchiveURL: "https://go.dev/dl/{{.Asset}}",
},
```

Example (Node.js LTS):

```go
{
    Name: "node", DisplayName: "Node.js", Category: "Runtimes",
    Description: "JavaScript runtime (LTS)",
    VersionCmd:  []string{"node", "--version"},
    Kind:   KindArchive,
    Source: SourceNodeJS,
    AssetTemplate: "node-{{.Version}}-{{.OS}}-{{.Arch}}.tar.xz",
    ArchiveURL:    "https://nodejs.org/dist/{{.Version}}/{{.Asset}}",
    ChecksumsURL:  "https://nodejs.org/dist/{{.Version}}/SHASUMS256.txt",
    BinRel:        "bin",
},
```

### `KindScript` — official installer script

Downloads the official script and runs it with `bash`, with configurable
arguments and environment (templated with `{{.Prefix}}` etc). The script's
binaries are symlinked from `BinRel`. No version tracking (updates are handled
by the tool itself).

```go
{
    Name: "bun", DisplayName: "Bun", Category: "Runtimes",
    Description: "Fast JavaScript runtime",
    VersionCmd:  []string{"bun", "--version"},
    Kind:   KindScript,
    ScriptURL: "https://bun.sh/install",
    ScriptEnv:  []string{"BUN_INSTALL={{.Prefix}}/bun"},
    BinRel:     "bun/bin",
    Dirs:       []string{"bun"}, // removed on uninstall
},
```

`Dirs` lists prefix-relative directories deleted on uninstall (defaults to
`tools/<name>` for archives). Set it for script installs that write outside
`tools/` (Rust uses `rustup` + `cargo`).

### `KindPkg` — system package

Installed via the distro's native package manager with sudo (layer chosen
automatically from `/etc/os-release`). `Pkg` maps manager ids to package
names — if the current system's manager has no entry, the table shows
`unsupport` and install is refused.

```go
{
    Name: "ripgrep", DisplayName: "Ripgrep", Category: "Tools",
    Description: "Fast line-oriented search (system pkg)",
    VersionCmd:  []string{"rg", "--version"},
    Kind: KindPkg,
    Pkg: map[string]string{
        "apt": "ripgrep", "dnf": "ripgrep", "pacman": "ripgrep",
        "zypper": "ripgrep", "apk": "ripgrep",
    },
},
```

Supported managers: `apt`, `dnf`, `pacman`, `zypper`, `apk`. To support a new
manager, add a layer in `internal/installer/pkgmgr.go` (distro IDs it serves,
install/remove argument builders) — tools then only need a `Pkg` entry for it.

## Version formats

Pinned versions are normalized per source automatically:
`1.27.4` → `go1.27.4` (go.dev), `24.19.0` → `v24.19.0` (nodejs, github).
Comparisons are numeric (`catalog/compare.go`), so update detection works
across `go1.27.4`/`v24.19.0`/`0.64.1` styles.

## Validation & tests

Every entry is validated at startup (`catalog.ValidateAll()` — required
fields per kind, known sources, non-empty `Pkg` maps). Run:

```sh
go test ./...
go vet ./...
```

Add a case to `internal/catalog/parse_test.go` when you add a new source
selector or comparison edge, and to `internal/installer/pkgmgr_test.go` when
you add a package-manager layer.
