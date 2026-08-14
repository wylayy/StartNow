package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"startnow/internal/installer"
)

func normalizeVersion(t *Tool, v string) string {
	if v == "" {
		return ""
	}
	switch t.Source {
	case SourceGoDev:
		if !strings.HasPrefix(v, "go") {
			return "go" + v
		}
	case SourceNodeJS, SourceGitHub:
		if !strings.HasPrefix(v, "v") {
			return "v" + v
		}
	}
	return v
}

// Install installs t and returns the resolved version ("" when the tool has
// no tracked version, e.g. script installs).
func Install(t *Tool, env *installer.Env) (string, error) {
	t.Version = normalizeVersion(t, t.Version)
	switch t.Kind {
	case KindArchive:
		return installArchive(t, env)
	case KindScript:
		return "", installScript(t, env)
	case KindPkg:
		return installPkg(t, env)
	}
	return "", fmt.Errorf("%s: unsupported kind %q", t.Name, t.Kind)
}

// ResolveVersion returns the latest (or pinned) version for t without
// installing anything. Script and pkg installs have no tracked version.
func ResolveVersion(t *Tool, env *installer.Env) (string, error) {
	if t.Kind != KindArchive {
		return "", fmt.Errorf("%s: no version tracking for this install kind", t.Name)
	}
	_, _, _, version, err := resolveArchive(t, env)
	return version, err
}

// ResolveDownload returns the concrete download URL for t (all template
// placeholders expanded). Only archive tools have a download URL.
func ResolveDownload(t *Tool, env *installer.Env) (string, error) {
	if t.Kind != KindArchive {
		return "", fmt.Errorf("%s: no download URL for this install kind", t.Name)
	}
	_, url, _, _, err := resolveArchive(t, env)
	return url, err
}

func installArchive(t *Tool, env *installer.Env) (string, error) {
	asset, url, checksum, version, err := resolveArchive(t, env)
	if err != nil {
		return "", err
	}
	archive, err := env.Download(t.Name, asset, url)
	if err != nil {
		return "", err
	}
	if err := env.VerifySHA256(t.Name, archive, checksum); err != nil {
		return "", err
	}
	dir := filepath.Join(env.Prefix, "tools", t.Name)
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := env.Extract(t.Name, archive, dir); err != nil {
		return "", err
	}
	bin := dir
	if t.BinRel != "" {
		bin = filepath.Join(dir, filepath.FromSlash(t.BinRel))
	}
	if err := env.LinkIntoBin(t.Name, bin); err != nil {
		return "", err
	}
	return version, nil
}

func installScript(t *Tool, env *installer.Env) error {
	env.Report(t.Name, installer.StepResolving, 0, "downloading installer script")
	script, err := env.Download(t.Name, t.Name+"-install.sh", t.ScriptURL)
	if err != nil {
		return err
	}
	env.Report(t.Name, installer.StepInstalling, 0, "running installer script")
	data := templateData(env, t, "", "", "")
	var args, extraEnv []string
	for _, a := range t.ScriptArgs {
		args = append(args, expand(a, data))
	}
	for _, e := range t.ScriptEnv {
		extraEnv = append(extraEnv, expand(e, data))
	}
	if err := env.RunScript(t.Name, script, args, extraEnv); err != nil {
		return err
	}
	if t.BinRel == "" {
		return nil
	}
	return env.LinkIntoBin(t.Name, filepath.Join(env.Prefix, filepath.FromSlash(t.BinRel)))
}

// Uninstall removes t from the install prefix and the manifest.
func Uninstall(t *Tool, env *installer.Env) error {
	if t.Kind == KindPkg {
		if err := removePkg(t, env); err != nil {
			return err
		}
		env.Report(t.Name, installer.StepDone, 1, "uninstalled")
		return env.ForgetInstall(t.Name)
	}
	dirs := t.Dirs
	if len(dirs) == 0 {
		dirs = []string{filepath.Join("tools", t.Name)}
	}
	for _, d := range dirs {
		abs := filepath.Join(env.Prefix, filepath.FromSlash(d))
		if err := env.RemoveLinksUnder(abs); err != nil {
			return err
		}
		if err := os.RemoveAll(abs); err != nil {
			return err
		}
	}
	env.Report(t.Name, installer.StepDone, 1, "uninstalled")
	return env.ForgetInstall(t.Name)
}
