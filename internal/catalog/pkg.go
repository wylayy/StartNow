package catalog

import (
	"fmt"

	"startnow/internal/installer"
)

func installPkg(t *Tool, env *installer.Env) (string, error) {
	mgr := env.PkgMgr
	if mgr == nil {
		return "", fmt.Errorf("%s: no package manager detected on this system", t.Name)
	}
	pkg, ok := t.Pkg[mgr.ID]
	if !ok || pkg == "" {
		return "", fmt.Errorf("%s: no package name for manager %q", t.Name, mgr.ID)
	}
	args := mgr.Install(pkg)
	prefix, err := env.SudoCommand()
	if err != nil {
		return "", err
	}
	env.Report(t.Name, installer.StepResolving, 0, fmt.Sprintf("installing %s via %s", pkg, mgr.Name))
	if err := env.RunCommand(t.Name, append(prefix, args...)...); err != nil {
		return "", err
	}
	env.Report(t.Name, installer.StepDone, 1, fmt.Sprintf("installed %s via %s", pkg, mgr.Name))
	return "", nil
}

func removePkg(t *Tool, env *installer.Env) error {
	mgr := env.PkgMgr
	if mgr == nil {
		return fmt.Errorf("%s: no package manager detected on this system", t.Name)
	}
	pkg, ok := t.Pkg[mgr.ID]
	if !ok || pkg == "" {
		return fmt.Errorf("%s: no package name for manager %q", t.Name, mgr.ID)
	}
	args := mgr.Remove(pkg)
	prefix, err := env.SudoCommand()
	if err != nil {
		return err
	}
	return env.RunCommand(t.Name, append(prefix, args...)...)
}
