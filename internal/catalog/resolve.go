package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"startnow/internal/installer"
)

// archiveResolver resolves the concrete asset (name, download URL, sha256)
// and resolved version for an archive tool.
type archiveResolver func(t *Tool, env *installer.Env) (asset, url, checksum, version string, err error)

var archiveResolvers = map[Source]archiveResolver{
	SourceGoDev:  resolveGoDev,
	SourceNodeJS: resolveNodeJS,
	SourceGitHub: resolveGitHub,
}

func resolveArchive(t *Tool, env *installer.Env) (string, string, string, string, error) {
	env.Report(t.Name, installer.StepResolving, 0, "resolving version")
	r, ok := archiveResolvers[t.Source]
	if !ok {
		return "", "", "", "", fmt.Errorf("%s: no resolver for source %q", t.Name, t.Source)
	}
	return r(t, env)
}

func expand(s string, data map[string]string) string {
	for k, v := range data {
		s = strings.ReplaceAll(s, "{{."+k+"}}", v)
	}
	return s
}

func templateData(env *installer.Env, t *Tool, version, tag, asset string) map[string]string {
	return map[string]string{
		"Name":    t.Name,
		"Version": version,
		"Tag":     tag,
		"Asset":   asset,
		"OS":      goOS(),
		"Arch":    goArch(),
		"Repo":    t.Repo,
		"Prefix":  env.Prefix,
		"BinDir":  env.BinDir,
	}
}

// go.dev source.

type goRelease struct {
	Version string   `json:"version"`
	Files   []goFile `json:"files"`
}

type goFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Kind     string `json:"kind"`
	SHA256   string `json:"sha256"`
}

var goStableRE = regexp.MustCompile(`^go\d+\.\d+\.\d+$`)

func selectGoFile(releases []goRelease, os_, arch string) (goFile, error) {
	for _, r := range releases {
		if !goStableRE.MatchString(r.Version) {
			continue
		}
		for _, f := range r.Files {
			if f.OS == os_ && f.Arch == arch && f.Kind == "archive" && f.SHA256 != "" {
				return f, nil
			}
		}
	}
	return goFile{}, fmt.Errorf("no stable Go archive for %s/%s", os_, arch)
}

func findGoFile(releases []goRelease, version, os_, arch string) (goFile, error) {
	for _, r := range releases {
		if r.Version != version {
			continue
		}
		for _, f := range r.Files {
			if f.OS == os_ && f.Arch == arch && f.Kind == "archive" && f.SHA256 != "" {
				return f, nil
			}
		}
	}
	return goFile{}, fmt.Errorf("no Go archive %s for %s/%s", version, os_, arch)
}

func resolveGoDev(t *Tool, env *installer.Env) (string, string, string, string, error) {
	var releases []goRelease
	if err := env.FetchJSON("https://go.dev/dl/?mode=json&include=all", &releases); err != nil {
		return "", "", "", "", err
	}
	var f goFile
	var version string
	var err error
	if t.Version != "" {
		f, err = findGoFile(releases, t.Version, goOS(), goArch())
		version = t.Version
	} else {
		f, err = selectGoFile(releases, goOS(), goArch())
	}
	if err != nil {
		return "", "", "", "", err
	}
	if version == "" {
		for _, rel := range releases {
			if goStableRE.MatchString(rel.Version) {
				version = rel.Version
				break
			}
		}
	}
	env.Report(t.Name, installer.StepResolving, 1, "resolved: "+f.Filename)
	data := templateData(env, t, "", "", f.Filename)
	return f.Filename, expand(t.ArchiveURL, data), f.SHA256, version, nil
}

// nodejs.org source.

type nodeEntry struct {
	Version string `json:"version"`
	LTS     any    `json:"lts"`
}

func selectNodeLTS(entries []nodeEntry) (nodeEntry, error) {
	for _, e := range entries {
		if s, ok := e.LTS.(string); ok && s != "" {
			return e, nil
		}
	}
	return nodeEntry{}, fmt.Errorf("no Node.js LTS release found")
}

func nodePlatform() (string, string) {
	archName := "x64"
	if goArch() == "arm64" {
		archName = "arm64"
	}
	return "linux", archName
}

func resolveNodeJS(t *Tool, env *installer.Env) (string, string, string, string, error) {
	version := t.Version
	if version == "" {
		var entries []nodeEntry
		if err := env.FetchJSON("https://nodejs.org/dist/index.json", &entries); err != nil {
			return "", "", "", "", err
		}
		e, err := selectNodeLTS(entries)
		if err != nil {
			return "", "", "", "", err
		}
		version = e.Version
	}
	osName, archName := nodePlatform()
	data := templateData(env, t, version, "", "")
	data["OS"] = osName
	data["Arch"] = archName
	asset := expand(t.AssetTemplate, data)
	data["Asset"] = asset
	url := expand(t.ArchiveURL, data)
	sumsBody, err := env.Get(expand(t.ChecksumsURL, data))
	if err != nil {
		return "", "", "", "", err
	}
	want, err := sha256FromSums(string(sumsBody), asset)
	if err != nil {
		return "", "", "", "", err
	}
	env.Report(t.Name, installer.StepResolving, 1, "resolved: "+asset)
	return asset, url, want, version, nil
}

func sha256FromSums(sums, filename string) (string, error) {
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", filename)
}

// github source.

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var osVariants = map[string][]string{
	"linux": {"linux", "Linux"},
}

var archVariants = map[string][]string{
	"amd64": {"x86_64", "64-bit", "amd64"},
	"arm64": {"arm64", "aarch64"},
}

func githubAssetCandidates(t *Tool, version string) []string {
	prefix := expand(t.AssetTemplate, map[string]string{"Version": version})
	var out []string
	for _, osV := range osVariants[goOS()] {
		for _, archV := range archVariants[goArch()] {
			for _, ext := range []string{".tar.gz", ".zip"} {
				out = append(out, fmt.Sprintf("%s_%s_%s%s", prefix, osV, archV, ext))
			}
		}
	}
	return out
}

func checksumAssets(t *Tool) []string {
	if len(t.ChecksumAsset) > 0 {
		return t.ChecksumAsset
	}
	return []string{"checksums.txt", "SHA256SUMS", "SHA256SUMS.txt"}
}

func resolveGitHub(t *Tool, env *installer.Env) (string, string, string, string, error) {
	apiURL := "https://api.github.com/repos/" + t.Repo + "/releases/latest"
	if t.Version != "" {
		apiURL = "https://api.github.com/repos/" + t.Repo + "/releases/tags/" + t.Version
	}
	var rel ghRelease
	if err := env.FetchJSON(apiURL, &rel); err != nil {
		return "", "", "", "", err
	}
	tag := rel.TagName
	version := strings.TrimPrefix(tag, "v")
	candidates := githubAssetCandidates(t, version)
	var asset, sumsAsset, sumsURL string
	for _, a := range rel.Assets {
		if asset == "" {
			for _, c := range candidates {
				if a.Name == c {
					asset = c
					break
				}
			}
		}
		if sumsAsset == "" {
			for _, c := range checksumAssets(t) {
				if a.Name == c {
					sumsAsset = c
					sumsURL = a.BrowserDownloadURL
					break
				}
			}
		}
	}
	if asset == "" {
		return "", "", "", "", fmt.Errorf("no asset matching %v in release %s", candidates, tag)
	}
	if sumsAsset == "" {
		return "", "", "", "", fmt.Errorf("checksum asset not found in release %s", tag)
	}
	sumsBody, err := env.Get(sumsURL)
	if err != nil {
		return "", "", "", "", err
	}
	want, err := sha256FromSums(string(sumsBody), asset)
	if err != nil {
		return "", "", "", "", err
	}
	env.Report(t.Name, installer.StepResolving, 1, "resolved: "+tag)
	data := templateData(env, t, version, tag, asset)
	return asset, expand(t.ArchiveURL, data), want, version, nil
}
