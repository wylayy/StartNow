package machine

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Hostname string
	User     string
	OS       string
	Kernel   string
	Arch     string
	CPUs     int
	CPUModel string
	MemTotal uint64
	MemAvail uint64
	Uptime   time.Duration
	Shell    string
	GoVer    string
}

func Collect() Info {
	inf := Info{
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
		GoVer: runtime.Version(),
		Shell: os.Getenv("SHELL"),
		CPUs:  runtime.NumCPU(),
	}
	if h, err := os.Hostname(); err == nil {
		inf.Hostname = h
	}
	if u, err := user.Current(); err == nil {
		inf.User = u.Username
	}
	if runtime.GOOS == "linux" {
		inf.Kernel = readFirst("/proc/sys/kernel/osrelease")
		if model, cpus := parseCPUInfo(readFirst("/proc/cpuinfo")); cpus > 0 {
			inf.CPUModel, inf.CPUs = model, cpus
		}
		if total, avail := parseMemInfo(readFirst("/proc/meminfo")); total > 0 {
			inf.MemTotal, inf.MemAvail = total, avail
		}
		inf.Uptime = parseUptime(readFirst("/proc/uptime"))
	}
	return inf
}

func readFirst(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func parseCPUInfo(content string) (model string, cpus int) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "processor"):
			cpus++
		case strings.HasPrefix(line, "model name"):
			if i := strings.IndexByte(line, ':'); i >= 0 {
				model = strings.TrimSpace(line[i+1:])
			}
		}
	}
	return model, cpus
}

func parseMemInfo(content string) (total, avail uint64) {
	t, a, _, _ := parseMemInfoFull(content)
	return t, a
}

func parseUptime(s string) time.Duration {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

func HumanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func HumanDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
