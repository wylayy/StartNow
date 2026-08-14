package machine

import "testing"

func TestParseOSReleaseArch(t *testing.T) {
	content := `NAME="Arch Linux"
PRETTY_NAME="Arch Linux"
ID=arch
BUILD_ID=rolling
ANSI_COLOR="38;2;23;147;209"
HOME_URL="https://archlinux.org/"
`
	d := parseOSRelease(content)
	if d.ID != "arch" || d.Name != "Arch Linux" || d.Pretty != "Arch Linux" {
		t.Errorf("got %+v", d)
	}
	if pmByID[d.ID] != "pacman" {
		t.Errorf("arch should map to pacman, got %q", pmByID[d.ID])
	}
}

func TestParseOSReleaseFedora(t *testing.T) {
	content := `NAME="Fedora Linux"
VERSION="40 (KDE Plasma)"
ID=fedora
VERSION_ID=40
VERSION_CODENAME=""
PRETTY_NAME="Fedora Linux 40 (KDE Plasma)"
`
	d := parseOSRelease(content)
	if d.ID != "fedora" || d.Version != "40" || d.Pretty != "Fedora Linux 40 (KDE Plasma)" {
		t.Errorf("got %+v", d)
	}
	if pmByID[d.ID] != "dnf" {
		t.Errorf("fedora should map to dnf, got %q", pmByID[d.ID])
	}
}

func TestParseOSReleaseUbuntu(t *testing.T) {
	content := `PRETTY_NAME="Ubuntu 24.04.1 LTS"
NAME="Ubuntu"
VERSION_ID="24.04"
VERSION="24.04.1 LTS (Noble Numbat)"
ID=ubuntu
`
	d := parseOSRelease(content)
	if d.ID != "ubuntu" || d.Version != "24.04" {
		t.Errorf("got %+v", d)
	}
	if pmByID[d.ID] != "apt" {
		t.Errorf("ubuntu should map to apt, got %q", pmByID[d.ID])
	}
}

func TestParseOSReleaseEmpty(t *testing.T) {
	if d := parseOSRelease(""); d.ID != "" {
		t.Errorf("expected empty, got %+v", d)
	}
}
