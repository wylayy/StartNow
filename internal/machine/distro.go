package machine

import (
	"os"
	"os/exec"
	"strings"
)

type Distro struct {
	ID             string // "arch", "fedora", "debian", ...
	Name           string // "Arch Linux", "Fedora Linux"
	Version        string // "40", "24.04", ...
	Pretty         string // "Fedora Linux 40 (KDE Plasma)"
	PackageManager string // "pacman", "dnf", "apt", ...
}

var pmByID = map[string]string{
	"debian": "apt", "ubuntu": "apt", "linuxmint": "apt", "pop": "apt", "raspbian": "apt", "elementary": "apt",
	"fedora": "dnf", "rhel": "dnf", "centos": "dnf", "rocky": "dnf", "almalinux": "dnf", "ol": "dnf",
	"arch": "pacman", "manjaro": "pacman", "endeavouros": "pacman", "cachyos": "pacman",
	"opensuse": "zypper", "sles": "zypper",
	"alpine": "apk",
}

// DetectDistro identifies the running Linux distribution and its native
// package manager. Non-Linux systems yield an empty Distro.
func DetectDistro() Distro {
	d := parseOSRelease(readFirst("/etc/os-release"))
	if d.ID == "" {
		switch {
		case exists("/etc/arch-release"):
			d.ID, d.Name = "arch", "Arch Linux"
		case exists("/etc/fedora-release"):
			d.ID, d.Name = "fedora", "Fedora Linux"
		case exists("/etc/debian_version"):
			d.ID, d.Name = "debian", "Debian GNU/Linux"
		case exists("/etc/alpine-release"):
			d.ID, d.Name = "alpine", "Alpine Linux"
		}
	}
	if d.Pretty == "" {
		d.Pretty = strings.TrimSpace(d.Name + " " + d.Version)
	}
	d.PackageManager = pmByID[d.ID]
	if d.PackageManager == "" {
		for _, pm := range []string{"apt", "dnf", "pacman", "zypper", "apk"} {
			if _, err := exec.LookPath(pm); err == nil {
				d.PackageManager = pm
				break
			}
		}
	}
	return d
}

func parseOSRelease(content string) Distro {
	var d Distro
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		switch strings.TrimSpace(key) {
		case "ID":
			d.ID = value
		case "NAME":
			d.Name = value
		case "VERSION_ID":
			d.Version = value
		case "PRETTY_NAME":
			d.Pretty = value
		}
	}
	return d
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
