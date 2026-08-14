package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const manifestName = "manifest.json"

type InstallEntry struct {
	Version string    `json:"version,omitempty"`
	Time    time.Time `json:"time,omitempty"`
}

type Manifest struct {
	Installs map[string]InstallEntry `json:"installs"`
}

func (e *Env) manifestPath() string {
	return filepath.Join(e.Prefix, manifestName)
}

func (e *Env) LoadManifest() (*Manifest, error) {
	m := &Manifest{Installs: map[string]InstallEntry{}}
	b, err := os.ReadFile(e.manifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, fmt.Errorf("corrupt manifest: %w", err)
	}
	if m.Installs == nil {
		m.Installs = map[string]InstallEntry{}
	}
	return m, nil
}

func (e *Env) SaveManifest(m *Manifest) error {
	if m.Installs == nil {
		m.Installs = map[string]InstallEntry{}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := e.manifestPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, e.manifestPath())
}

func (e *Env) RecordInstall(name, version string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	m, err := e.LoadManifest()
	if err != nil {
		return err
	}
	m.Installs[name] = InstallEntry{Version: version, Time: time.Now()}
	return e.SaveManifest(m)
}

func (e *Env) ForgetInstall(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	m, err := e.LoadManifest()
	if err != nil {
		return err
	}
	delete(m.Installs, name)
	return e.SaveManifest(m)
}

// RemoveLinksUnder removes symlinks in BinDir whose target lives under dir.
func (e *Env) RemoveLinksUnder(dir string) error {
	entries, err := os.ReadDir(e.BinDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	target := filepath.Clean(dir)
	prefix := target + string(filepath.Separator)
	for _, ent := range entries {
		if ent.Type()&os.ModeSymlink == 0 {
			continue
		}
		link := filepath.Join(e.BinDir, ent.Name())
		dest, err := os.Readlink(link)
		if err != nil {
			continue
		}
		if dest == target || filepath.HasPrefix(filepath.Clean(dest), prefix) {
			if err := os.Remove(link); err != nil {
				return err
			}
		}
	}
	return nil
}
