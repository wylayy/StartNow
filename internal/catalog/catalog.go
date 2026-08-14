package catalog

import (
	"fmt"
	"runtime"

	"startnow/internal/installer"
)

type Kind string

const (
	KindArchive Kind = "archive"
	KindScript  Kind = "script"
	KindPkg     Kind = "pkg"
)

type Source string

const (
	SourceGoDev  Source = "go.dev"
	SourceNodeJS Source = "nodejs.org"
	SourceGitHub Source = "github"
)

type Tool struct {
	Name        string
	DisplayName string
	Description string
	Category    string
	VersionCmd  []string

	Kind   Kind
	Source Source

	// BinRel is the path relative to the install prefix that contains the
	// binaries to link into ~/.startnow/bin. Empty for KindArchive means the
	// binaries sit at the archive root.
	BinRel string

	// Archive metadata (KindArchive). Templates support the placeholders
	// {{.Name}} {{.Version}} {{.Tag}} {{.Asset}} {{.OS}} {{.Arch}} {{.Prefix}} {{.BinDir}}.
	AssetTemplate string   // asset name template (nodejs, github)
	ArchiveURL    string   // full download URL template
	ChecksumsURL  string   // checksum list URL template (nodejs)
	ChecksumAsset []string // checksum asset names (github)
	Repo          string   // "owner/repo" (github)

	// Script metadata (KindScript).
	ScriptURL  string
	ScriptArgs []string
	ScriptEnv  []string

	// Pkg maps a package-manager id ("apt", "dnf", "pacman", "zypper",
	// "apk") to the native package name (KindPkg).
	Pkg map[string]string

	// Dirs lists prefix-relative directories to remove on uninstall.
	// Empty for KindArchive defaults to "tools/<name>".
	Dirs []string

	// Optional pinned version; empty means resolve the latest.
	Version string
}

func Tools() []Tool {
	return []Tool{
		{
			Name: "go", DisplayName: "Go", Category: "Languages", Description: "Go toolchain",
			VersionCmd: []string{"go", "version"},
			Kind:       KindArchive, Source: SourceGoDev,
			ArchiveURL: "https://go.dev/dl/{{.Asset}}",
			BinRel:     "bin",
		},
		{
			Name: "node", DisplayName: "Node.js", Category: "Runtimes", Description: "JavaScript runtime (LTS)",
			VersionCmd: []string{"node", "--version"},
			Kind:       KindArchive, Source: SourceNodeJS,
			AssetTemplate: "node-{{.Version}}-{{.OS}}-{{.Arch}}.tar.xz",
			ArchiveURL:    "https://nodejs.org/dist/{{.Version}}/{{.Asset}}",
			ChecksumsURL:  "https://nodejs.org/dist/{{.Version}}/SHASUMS256.txt",
			BinRel:        "bin",
		},
		{
			Name: "rust", DisplayName: "Rust", Category: "Languages", Description: "Rust toolchain via rustup",
			VersionCmd: []string{"rustc", "--version"},
			Kind:       KindScript,
			ScriptURL:  "https://sh.rustup.rs",
			ScriptArgs: []string{"-y", "--no-modify-path", "--default-toolchain", "stable", "--profile", "minimal"},
			ScriptEnv:  []string{"RUSTUP_HOME={{.Prefix}}/rustup", "CARGO_HOME={{.Prefix}}/cargo"},
			BinRel:     "cargo/bin",
			Dirs:       []string{"rustup", "cargo"},
		},
		{
			Name: "bun", DisplayName: "Bun", Category: "Runtimes", Description: "Fast JavaScript runtime",
			VersionCmd: []string{"bun", "--version"},
			Kind:       KindScript,
			ScriptURL:  "https://bun.sh/install",
			ScriptEnv:  []string{"BUN_INSTALL={{.Prefix}}/bun"},
			BinRel:     "bun/bin",
			Dirs:       []string{"bun"},
		},
		{
			Name: "lazygit", DisplayName: "LazyGit", Category: "Tools", Description: "Terminal UI for git",
			VersionCmd: []string{"lazygit", "--version"},
			Kind:       KindArchive, Source: SourceGitHub,
			Repo:          "jesseduffield/lazygit",
			AssetTemplate: "lazygit_{{.Version}}",
			ArchiveURL:    "https://github.com/{{.Repo}}/releases/download/{{.Tag}}/{{.Asset}}",
			ChecksumAsset: []string{"checksums.txt", "SHA256SUMS"},
		},
		{
			Name: "ripgrep", DisplayName: "Ripgrep", Category: "Tools", Description: "Fast line-oriented search (system pkg)",
			VersionCmd: []string{"rg", "--version"},
			Kind:       KindPkg,
			Pkg: map[string]string{
				"apt": "ripgrep", "dnf": "ripgrep", "pacman": "ripgrep", "zypper": "ripgrep", "apk": "ripgrep",
			},
		},
		{
			Name: "htop", DisplayName: "Htop", Category: "Tools", Description: "Interactive process viewer (system pkg)",
			VersionCmd: []string{"htop", "--version"},
			Kind:       KindPkg,
			Pkg: map[string]string{
				"apt": "htop", "dnf": "htop", "pacman": "htop", "zypper": "htop", "apk": "htop",
			},
		},
	}
}

func Validate(t *Tool) error {
	if t.Name == "" || t.DisplayName == "" {
		return fmt.Errorf("tool missing name or display name")
	}
	switch t.Kind {
	case KindScript:
		if t.ScriptURL == "" {
			return fmt.Errorf("%s: script kind requires ScriptURL", t.Name)
		}
	case KindPkg:
		if len(t.Pkg) == 0 {
			return fmt.Errorf("%s: pkg kind requires Pkg package names", t.Name)
		}
	case KindArchive:
		if t.Source == "" {
			return fmt.Errorf("%s: archive kind requires Source", t.Name)
		}
		if _, ok := archiveResolvers[t.Source]; !ok {
			return fmt.Errorf("%s: unknown archive source %q", t.Name, t.Source)
		}
		if t.ArchiveURL == "" {
			return fmt.Errorf("%s: archive kind requires ArchiveURL", t.Name)
		}
		switch t.Source {
		case SourceNodeJS:
			if t.AssetTemplate == "" || t.ChecksumsURL == "" {
				return fmt.Errorf("%s: nodejs source requires AssetTemplate and ChecksumsURL", t.Name)
			}
		case SourceGitHub:
			if t.Repo == "" || t.AssetTemplate == "" {
				return fmt.Errorf("%s: github source requires Repo and AssetTemplate", t.Name)
			}
		}
	default:
		return fmt.Errorf("%s: unknown kind %q", t.Name, t.Kind)
	}
	return nil
}

func ValidateAll() error {
	for _, t := range Tools() {
		if err := Validate(&t); err != nil {
			return err
		}
	}
	return nil
}

func goOS() string   { return runtime.GOOS }
func goArch() string { return runtime.GOARCH }

// Supported reports whether t can be installed on the current system.
// Archive and script tools always are; pkg tools need a detected package
// manager with a package name for this system.
func (t *Tool) Supported(env *installer.Env) bool {
	if t.Kind != KindPkg {
		return true
	}
	if env.PkgMgr == nil {
		return false
	}
	_, ok := t.Pkg[env.PkgMgr.ID]
	return ok
}
