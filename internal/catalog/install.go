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

func Install(t *Tool, env *installer.Env) error {
	t.Version = normalizeVersion(t, t.Version)
	switch t.Kind {
	case KindArchive:
		return installArchive(t, env)
	case KindScript:
		return installScript(t, env)
	}
	return fmt.Errorf("%s: unsupported kind %q", t.Name, t.Kind)
}

func installArchive(t *Tool, env *installer.Env) error {
	asset, url, checksum, err := resolveArchive(t, env)
	if err != nil {
		return err
	}
	archive, err := env.Download(t.Name, asset, url)
	if err != nil {
		return err
	}
	if err := env.VerifySHA256(t.Name, archive, checksum); err != nil {
		return err
	}
	dir := filepath.Join(env.Prefix, "tools", t.Name)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := env.Extract(t.Name, archive, dir); err != nil {
		return err
	}
	bin := dir
	if t.BinRel != "" {
		bin = filepath.Join(dir, filepath.FromSlash(t.BinRel))
	}
	return env.LinkIntoBin(t.Name, bin)
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
