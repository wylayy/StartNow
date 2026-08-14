package catalog

import "testing"

func TestSelectGoFileSkipsPrereleases(t *testing.T) {
	releases := []goRelease{
		{Version: "go1.28rc1", Files: []goFile{{Filename: "go1.28rc1.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "aaa"}}},
		{Version: "go1.27.4", Files: []goFile{
			{Filename: "go1.27.4.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "abc123"},
			{Filename: "go1.27.4.linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Kind: "archive", SHA256: "def456"},
		}},
	}
	f, err := selectGoFile(releases, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "go1.27.4.linux-amd64.tar.gz" || f.SHA256 != "abc123" {
		t.Errorf("got %+v", f)
	}
}

func TestSelectGoFileNone(t *testing.T) {
	if _, err := selectGoFile(nil, "linux", "amd64"); err == nil {
		t.Error("expected error")
	}
}

func TestFindGoFilePinnedVersion(t *testing.T) {
	releases := []goRelease{
		{Version: "go1.27.4", Files: []goFile{{Filename: "go1.27.4.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "abc"}}},
		{Version: "go1.26.5", Files: []goFile{{Filename: "go1.26.5.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "def"}}},
	}
	f, err := findGoFile(releases, "go1.26.5", "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if f.Filename != "go1.26.5.linux-amd64.tar.gz" {
		t.Errorf("got %s", f.Filename)
	}
	if _, err := findGoFile(releases, "go1.99.0", "linux", "amd64"); err == nil {
		t.Error("expected error for missing version")
	}
}

func TestSelectNodeLTSSkipsNonLTS(t *testing.T) {
	entries := []nodeEntry{
		{Version: "v25.0.0", LTS: false},
		{Version: "v24.8.0", LTS: "Krypton"},
		{Version: "v23.0.0", LTS: false},
	}
	e, err := selectNodeLTS(entries)
	if err != nil {
		t.Fatal(err)
	}
	if e.Version != "v24.8.0" {
		t.Errorf("got %s", e.Version)
	}
}

func TestSha256FromSums(t *testing.T) {
	sums := "aaa111  node-v24.8.0-linux-x64.tar.xz\nbbb222  node-v24.8.0-linux-arm64.tar.xz\n"
	want, err := sha256FromSums(sums, "node-v24.8.0-linux-x64.tar.xz")
	if err != nil {
		t.Fatal(err)
	}
	if want != "aaa111" {
		t.Errorf("got %s", want)
	}
	if _, err := sha256FromSums(sums, "missing"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestGitHubAssetCandidates(t *testing.T) {
	lg := Tool{Name: "lazygit", AssetTemplate: "lazygit_{{.Version}}"}
	cands := githubAssetCandidates(&lg, "0.64.1")
	contains := func(want string) bool {
		for _, c := range cands {
			if c == want {
				return true
			}
		}
		return false
	}
	if !contains("lazygit_0.64.1_linux_x86_64.tar.gz") {
		t.Errorf("missing new-style asset, got %v", cands)
	}
	if !contains("lazygit_0.64.1_Linux_64-bit.tar.gz") {
		t.Errorf("missing old-style asset, got %v", cands)
	}
}

func TestExpand(t *testing.T) {
	data := map[string]string{"Version": "v24.8.0", "Asset": "node-v24.8.0-linux-x64.tar.xz"}
	got := expand("https://nodejs.org/dist/{{.Version}}/{{.Asset}}", data)
	want := "https://nodejs.org/dist/v24.8.0/node-v24.8.0-linux-x64.tar.xz"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestValidateAll(t *testing.T) {
	if err := ValidateAll(); err != nil {
		t.Errorf("catalog invalid: %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []Tool{
		{Name: "x", DisplayName: "X", Kind: KindScript},
		{Name: "x", DisplayName: "X", Kind: KindArchive, Source: SourceGoDev},
		{Name: "x", DisplayName: "X", Kind: KindArchive, Source: "bogus", ArchiveURL: "u"},
		{Name: "x", DisplayName: "X", Kind: KindArchive, Source: SourceNodeJS, ArchiveURL: "u"},
		{Name: "x", DisplayName: "X", Kind: KindArchive, Source: SourceGitHub, ArchiveURL: "u", AssetTemplate: "a"},
	}
	for i, tc := range cases {
		if err := Validate(&tc); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}
