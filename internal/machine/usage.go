package machine

import (
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

type Proc struct {
	PID  int
	Name string
	Mem  uint64
}

type Usage struct {
	CPU       float64
	MemTotal  uint64
	MemAvail  uint64
	SwapTotal uint64
	SwapUsed  uint64
	Load1     float64
	Load5     float64
	Load15    float64
	DiskTotal uint64
	DiskUsed  uint64
	Procs     []Proc
}

type Sampler struct {
	prevTotal uint64
	prevIdle  uint64
}

func NewSampler() *Sampler { return &Sampler{} }

func (s *Sampler) Sample() Usage {
	u := Usage{}
	if runtime.GOOS != "linux" {
		return u
	}
	total, idle := parseCPUStat(readFirst("/proc/stat"))
	if s.prevTotal > 0 && total > s.prevTotal {
		dt := total - s.prevTotal
		di := idle - s.prevIdle
		if dt > 0 {
			u.CPU = (1 - float64(di)/float64(dt)) * 100
			if u.CPU < 0 {
				u.CPU = 0
			}
		}
	}
	s.prevTotal, s.prevIdle = total, idle
	memTotal, memAvail, swapTotal, swapFree := parseMemInfoFull(readFirst("/proc/meminfo"))
	u.MemTotal, u.MemAvail = memTotal, memAvail
	if swapTotal > swapFree {
		u.SwapTotal, u.SwapUsed = swapTotal, swapTotal-swapFree
	}
	u.Load1, u.Load5, u.Load15 = parseLoadAvg(readFirst("/proc/loadavg"))
	u.DiskTotal, u.DiskUsed = diskUsage("/")
	u.Procs = topProcs(8)
	return u
}

func parseCPUStat(content string) (total, idle uint64) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		if len(fields) < 5 {
			return 0, 0
		}
		var vals [8]uint64
		for i, f := range fields {
			if i >= len(vals) {
				break
			}
			vals[i], _ = strconv.ParseUint(f, 10, 64)
		}
		idle = vals[3] + vals[4]
		for _, v := range vals {
			total += v
		}
		return total, idle
	}
	return 0, 0
}

func parseMemInfoFull(content string) (total, avail, swapTotal, swapFree uint64) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		kb *= 1024
		switch fields[0] {
		case "MemTotal:":
			total = kb
		case "MemAvailable:":
			avail = kb
		case "SwapTotal:":
			swapTotal = kb
		case "SwapFree:":
			swapFree = kb
		}
	}
	return total, avail, swapTotal, swapFree
}

func parseLoadAvg(content string) (l1, l5, l15 float64) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

func diskUsage(path string) (total, used uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	bs := uint64(st.Bsize)
	total = st.Blocks * bs
	used = (st.Blocks - st.Bavail) * bs
	return total, used
}

func topProcs(n int) []Proc {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	page := uint64(os.Getpagesize())
	var procs []Proc
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		name := strings.TrimSpace(readFirst("/proc/" + e.Name() + "/comm"))
		if name == "" {
			continue
		}
		procs = append(procs, Proc{
			PID:  pid,
			Name: name,
			Mem:  parseStatmPages(readFirst("/proc/"+e.Name()+"/statm")) * page,
		})
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].Mem > procs[j].Mem })
	if len(procs) > n {
		procs = procs[:n]
	}
	return procs
}

func parseStatmPages(content string) uint64 {
	fields := strings.Fields(content)
	if len(fields) < 2 {
		return 0
	}
	pages, _ := strconv.ParseUint(fields[1], 10, 64)
	return pages
}
