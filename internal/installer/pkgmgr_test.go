package installer

import (
	"testing"

	"startnow/internal/machine"
)

func TestDetectPkgManager(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"arch", "pacman"},
		{"endeavouros", "pacman"},
		{"fedora", "dnf"},
		{"rhel", "dnf"},
		{"debian", "apt"},
		{"ubuntu", "apt"},
		{"opensuse", "zypper"},
		{"alpine", "apk"},
		{"void", ""}, // unknown: falls back to PATH probe on linux
	}
	for _, c := range cases {
		mgr := DetectPkgManager(machine.Distro{ID: c.id})
		if mgr == nil {
			if c.want != "" {
				t.Errorf("%s: expected %s, got nil", c.id, c.want)
			}
			continue
		}
		if c.want == "" {
			continue // PATH fallback found something; acceptable
		}
		if mgr.ID != c.want {
			t.Errorf("%s: got %s, want %s", c.id, mgr.ID, c.want)
		}
	}
}

func TestPkgManagerArgs(t *testing.T) {
	cases := []struct {
		id, op, pkg string
		want        []string
	}{
		{"apt", "install", "ripgrep", []string{"apt-get", "install", "-y", "ripgrep"}},
		{"apt", "remove", "htop", []string{"apt-get", "remove", "-y", "htop"}},
		{"dnf", "remove", "htop", []string{"dnf", "remove", "-y", "htop"}},
		{"pacman", "install", "ripgrep", []string{"pacman", "-S", "--noconfirm", "ripgrep"}},
		{"pacman", "remove", "htop", []string{"pacman", "-Rns", "--noconfirm", "htop"}},
		{"zypper", "remove", "htop", []string{"zypper", "remove", "-y", "htop"}},
		{"apk", "install", "htop", []string{"apk", "add", "htop"}},
		{"apk", "remove", "htop", []string{"apk", "del", "htop"}},
	}
	for _, c := range cases {
		mgr := PkgManagerByID(c.id)
		if mgr == nil {
			t.Fatalf("%s: no manager", c.id)
		}
		var got []string
		if c.op == "install" {
			got = mgr.Install(c.pkg)
		} else {
			got = mgr.Remove(c.pkg)
		}
		if len(got) != len(c.want) {
			t.Errorf("%s %s: got %v, want %v", c.id, c.op, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s %s: got %v, want %v", c.id, c.op, got, c.want)
				break
			}
		}
	}
}
