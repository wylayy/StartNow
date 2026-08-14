package installer

import (
	"os/exec"
	"slices"

	"startnow/internal/machine"
)

// PkgManager is a package-manager layer: it knows which distros it serves,
// the command to run, and how to install/remove a package.
type PkgManager struct {
	ID      string // "apt"
	Name    string // "APT"
	Command string // executable used for the PATH fallback
	IDs     []string
	Install func(pkg string) []string
	Remove  func(pkg string) []string
}

var pkgManagers = []PkgManager{
	{
		ID: "apt", Name: "APT", Command: "apt-get",
		IDs:     []string{"debian", "ubuntu", "linuxmint", "pop", "raspbian", "elementary"},
		Install: func(p string) []string { return []string{"apt-get", "install", "-y", p} },
		Remove:  func(p string) []string { return []string{"apt-get", "remove", "-y", p} },
	},
	{
		ID: "dnf", Name: "DNF", Command: "dnf",
		IDs:     []string{"fedora", "rhel", "centos", "rocky", "almalinux", "ol"},
		Install: func(p string) []string { return []string{"dnf", "install", "-y", p} },
		Remove:  func(p string) []string { return []string{"dnf", "remove", "-y", p} },
	},
	{
		ID: "pacman", Name: "Pacman", Command: "pacman",
		IDs:     []string{"arch", "manjaro", "endeavouros", "cachyos"},
		Install: func(p string) []string { return []string{"pacman", "-S", "--noconfirm", p} },
		Remove:  func(p string) []string { return []string{"pacman", "-Rns", "--noconfirm", p} },
	},
	{
		ID: "zypper", Name: "Zypper", Command: "zypper",
		IDs:     []string{"opensuse", "sles"},
		Install: func(p string) []string { return []string{"zypper", "install", "-y", p} },
		Remove:  func(p string) []string { return []string{"zypper", "remove", "-y", p} },
	},
	{
		ID: "apk", Name: "APK", Command: "apk",
		IDs:     []string{"alpine"},
		Install: func(p string) []string { return []string{"apk", "add", p} },
		Remove:  func(p string) []string { return []string{"apk", "del", p} },
	},
}

// PkgManagerByID returns the manager with the given id, or nil.
func PkgManagerByID(id string) *PkgManager {
	for i := range pkgManagers {
		if pkgManagers[i].ID == id {
			return &pkgManagers[i]
		}
	}
	return nil
}

// DetectPkgManager selects the package-manager layer for the detected distro,
// falling back to whatever manager exists on PATH.
func DetectPkgManager(distro machine.Distro) *PkgManager {
	for i := range pkgManagers {
		if slices.Contains(pkgManagers[i].IDs, distro.ID) {
			return &pkgManagers[i]
		}
	}
	for i := range pkgManagers {
		if _, err := exec.LookPath(pkgManagers[i].Command); err == nil {
			return &pkgManagers[i]
		}
	}
	return nil
}
