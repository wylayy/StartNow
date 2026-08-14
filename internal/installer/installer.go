package installer

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"startnow/internal/machine"
)

type Step int

const (
	StepResolving Step = iota
	StepDownloading
	StepVerifying
	StepExtracting
	StepInstalling
	StepDone
	StepFailed
)

func (s Step) String() string {
	switch s {
	case StepResolving:
		return "resolving"
	case StepDownloading:
		return "downloading"
	case StepVerifying:
		return "verifying"
	case StepExtracting:
		return "extracting"
	case StepInstalling:
		return "installing"
	case StepDone:
		return "done"
	case StepFailed:
		return "failed"
	}
	return "pending"
}

type Event struct {
	Tool     string
	Step     Step
	Progress float64
	Message  string
	Version  string
}

type Env struct {
	Prefix string
	BinDir string
	Cache  string
	Send   func(Event)
	Distro machine.Distro
	Sudo   string
	PkgMgr *PkgManager

	mu sync.Mutex // guards manifest file access
}

func NewEnv(send func(Event)) (*Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	env := &Env{
		Prefix: filepath.Join(home, ".startnow"),
		Cache:  filepath.Join(home, ".cache", "startnow"),
		Send:   send,
		Distro: machine.DetectDistro(),
	}
	env.Sudo = env.sudoStatus()
	env.PkgMgr = DetectPkgManager(env.Distro)
	env.BinDir = filepath.Join(env.Prefix, "bin")
	for _, d := range []string{env.Prefix, env.BinDir, env.Cache} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	return env, nil
}

func (e *Env) Report(tool string, step Step, progress float64, msg string) {
	if e.Send != nil {
		e.Send(Event{Tool: tool, Step: step, Progress: progress, Message: msg})
	}
}

func (e *Env) httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "startnow/0.1")
	return http.DefaultClient.Do(req)
}

func (e *Env) Get(url string) ([]byte, error) {
	resp, err := e.httpGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (e *Env) FetchJSON(url string, v any) error {
	body, err := e.Get(url)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (e *Env) Download(tool, name, url string) (string, error) {
	resp, err := e.httpGet(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	dst := filepath.Join(e.Cache, name)
	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	total := resp.ContentLength
	var n int64
	last := time.Now()
	buf := make([]byte, 64*1024)
	for {
		nr, rerr := resp.Body.Read(buf)
		if nr > 0 {
			if _, werr := f.Write(buf[:nr]); werr != nil {
				f.Close()
				os.Remove(dst)
				return "", werr
			}
			n += int64(nr)
			if time.Since(last) > 100*time.Millisecond {
				last = time.Now()
				msg := fmt.Sprintf("downloading %s", name)
				var p float64
				if total > 0 {
					p = float64(n) / float64(total)
					msg = fmt.Sprintf("downloading %s (%.1f/%.1f MB)", name, float64(n)/1e6, float64(total)/1e6)
				}
				e.Report(tool, StepDownloading, p, msg)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(dst)
			return "", rerr
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(dst)
		return "", err
	}
	e.Report(tool, StepDownloading, 1, fmt.Sprintf("downloaded %s", name))
	return dst, nil
}

func (e *Env) VerifySHA256(tool, file, want string) error {
	e.Report(tool, StepVerifying, 0, "verifying sha256 checksum")
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", path.Base(file), got, want)
	}
	e.Report(tool, StepVerifying, 1, "checksum verified")
	return nil
}

func (e *Env) Extract(tool, archive, dst string) error {
	e.Report(tool, StepExtracting, 0, fmt.Sprintf("extracting %s", path.Base(archive)))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	if err := extractArchive(archive, dst); err != nil {
		return err
	}
	e.Report(tool, StepExtracting, 1, "extracted")
	return nil
}

func (e *Env) LinkIntoBin(tool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	linked := 0
	for _, ent := range entries {
		if ent.IsDir() || ent.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			return err
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		src := filepath.Join(dir, ent.Name())
		dst := filepath.Join(e.BinDir, ent.Name())
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("symlink %s: %w", dst, err)
		}
		linked++
	}
	e.Report(tool, StepDone, 1, fmt.Sprintf("linked %d binaries into %s", linked, e.BinDir))
	if linked == 0 {
		return fmt.Errorf("no executable binaries found in %s", dir)
	}
	return nil
}

func (e *Env) RunScript(tool, script string, args, extraEnv []string) error {
	return e.runStreamed(tool, "bash", append([]string{script}, args...), extraEnv)
}

// RunCommand runs an external command (e.g. a package manager) with output
// streamed as events.
func (e *Env) RunCommand(tool string, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}
	return e.runStreamed(tool, args[0], args[1:], nil)
}

func (e *Env) runStreamed(tool, name string, args, extraEnv []string) error {
	e.Report(tool, StepInstalling, 0, "running "+name)
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var scanErr error
	for _, r := range []io.Reader{stdout, stderr} {
		wg.Add(1)
		go func(r io.Reader) {
			defer wg.Done()
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				e.Report(tool, StepInstalling, 0, sc.Text())
			}
			if err := sc.Err(); err != nil {
				mu.Lock()
				scanErr = err
				mu.Unlock()
			}
		}(r)
	}
	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		return fmt.Errorf("installer script failed: %w", err)
	}
	if scanErr != nil {
		return fmt.Errorf("reading installer script output: %w", scanErr)
	}
	e.Report(tool, StepInstalling, 1, "installer script finished")
	return nil
}

func (e *Env) Probe(name string, args ...string) (string, bool) {
	// Prefer StartNow's own installs so the status column reflects the
	// managed version, falling back to whatever is on PATH.
	bin := filepath.Join(e.BinDir, name)
	if _, err := os.Stat(bin); err != nil {
		bin, err = exec.LookPath(name)
		if err != nil {
			return "", false
		}
	}
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "installed", true
	}
	return v, true
}

// OnPath reports whether BinDir is listed in the user's PATH.
func (e *Env) OnPath() bool {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == e.BinDir {
			return true
		}
	}
	return false
}

func ExtractFilename(url string) string {
	if i := strings.LastIndexByte(url, '?'); i >= 0 {
		url = url[:i]
	}
	return path.Base(url)
}
