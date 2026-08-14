# StartNow

A Ninite-style installer for developer tools, built as a terminal UI in Go.

Select tools, press enter, and StartNow downloads, verifies, and installs them
reliably — with checksum validation, version pinning, updates, uninstalls, and
a live system monitor. Linux only (Debian/Ubuntu, Fedora, Arch, openSUSE,
Alpine and friends).

```
    ____  _             _   _   _
   / ___|| |_ __ _ _ __| |_| \ | | _____      __
   \___ \| __/ _` | '__| __|  \| |/ _ \ \ /\ / /
    ___) | || (_| | |  | |_| |\  | (_) \ V  V /
   |____/ \__\__,_|_|   \__|_| \_|\___/ \_/\_/
```

## Features

- **Tools tab** — searchable table of dev tools with category, install
  status, and per-system support flags. Select with `space`, install with
  `enter`.
- **Reliable installs** — official sources only; every archive is verified
  against the publisher's sha256 checksum before extraction. Binaries land in
  `~/.startnow/bin` (no sudo needed for direct downloads).
- **Version control** — pin any version (`v`), check for updates (`u`),
  uninstall (`x`). Installs are tracked in `~/.startnow/manifest.json`.
- **System packages** — tools marked as system packages are installed through
  the distro's native package manager (APT, DNF, Pacman, Zypper, APK) via
  sudo, auto-detected from `/etc/os-release`.
- **Machine tab** — distro, kernel, CPU, memory, shell, package manager and
  sudo status.
- **Usage tab** — live monitor: overall + per-core CPU, memory/swap, load,
  network rates, disk, and top processes by CPU.
- **Mouse support** — click the tab pills, click the filter field, right-click
  a tool row for its details panel.

## Requirements

- Linux, Go 1.24+ to build
- A terminal with truecolor support recommended

## Build & Run

```sh
make build        # produces bin/startnow
make run          # go run ./cmd/startnow
make test         # unit tests
```

Add `~/.startnow/bin` to your PATH:

```sh
export PATH="$HOME/.startnow/bin:$PATH"
```

## Native Packages

Build a Debian package (Debian/Ubuntu/Mint, amd64 or arm64):

```sh
make deb VERSION=0.1.0
# built dist/startnow_0.1.0_amd64.deb
sudo apt install ./dist/startnow_0.1.0_amd64.deb
```

The package is a static, dependency-free binary installed to `/usr/bin/startnow`
(so it works on PATH out of the box), plus a man page, desktop entry and icon.
Other formats: build an RPM with `fpm -s dir -t rpm`, or write a PKGBUILD
(`pkgver=0.1.0`, `arch=('x86_64')`, `make deb` layout) for Arch.

## Keys

| Key          | Action                                  |
| ------------ | --------------------------------------- |
| `↑/↓` `j/k`  | move in the table                       |
| `space`      | toggle selection                        |
| `a`          | select all / clear                      |
| `/`          | focus the filter (or click it)          |
| `v`          | set a pinned version                    |
| `u`          | check for / install updates             |
| `x`          | uninstall (managed tools)               |
| `i`          | tool details (or right-click a row)     |
| `enter`      | install selected tools                  |
| `tab` / `1-3`| switch tabs                             |
| `?`          | full help                               |
| `q` / `ctrl+c` | quit                                 |

## Architecture

```
cmd/startnow/          entry point (Linux-only guard)
internal/catalog/      tool definitions (metadata) + install engine
  catalog.go           Tool struct, Tools(), Validate(), Supported()
  install.go           Install / ResolveVersion / Uninstall dispatch
  resolve.go           go.dev, nodejs.org, github version resolvers
  pkg.go               system-package installs via the manager layer
  compare.go           numeric version comparison
internal/installer/    reliability engine
  installer.go         downloads, checksums, extraction, symlinks, events
  pkgmgr.go            package-manager layer abstraction (apt/dnf/…)
  sudo.go              privilege handling
  manifest.go          install tracking (~/.startnow/manifest.json)
internal/machine/      distro detection + system info + usage sampling
internal/ui/           Bubble Tea v2 UI (bubbles components, box-cli-maker)
```

Adding a new tool is metadata-only — see
[docs/adding-tools.md](docs/adding-tools.md).

## Layout

| Path                 | Purpose                                  |
| -------------------- | ---------------------------------------- |
| `~/.startnow/bin`    | symlinks to installed binaries (add to PATH) |
| `~/.startnow/tools/` | extracted archives (go, node, lazygit)   |
| `~/.startnow/rustup`, `~/.startnow/cargo`, `~/.startnow/bun` | script-installer homes |
| `~/.startnow/manifest.json` | what StartNow installed and when  |
| `~/.cache/startnow/` | downloaded archives (reused on re-install) |

## License

MIT
