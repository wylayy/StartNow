package machine

import (
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Proc struct {
	PID  int
	Name string
	Mem  uint64
	CPU  float64 // percent since the last sample
}

type Usage struct {
	CPU         float64
	Cores       []float64 // per-core CPU percent, capped at 8, sorted desc
	MemTotal    uint64
	MemAvail    uint64
	SwapTotal   uint64
	SwapUsed    uint64
	Load1       float64
	Load5       float64
	Load15      float64
	ProcRunning int
	ProcTotal   int
	NetRx       float64 // bytes/sec since the last sample
	NetTx       float64
	DiskTotal   uint64
	DiskUsed    uint64
	Procs       []Proc
}

const maxCores = 8

type Sampler struct {
	prevTotal uint64
	prevIdle  uint64

	prevCores map[int]coreStat

	prevNet     map[string]netStat
	prevNetTime time.Time

	prevProc     map[int]uint64 // pid -> utime+stime ticks
	prevProcTime time.Time
}

type coreStat struct {
	total uint64
	idle  uint64
}

type netStat struct {
	rx uint64
	tx uint64
}

func NewSampler() *Sampler {
	return &Sampler{
		prevCores: map[int]coreStat{},
		prevNet:   map[string]netStat{},
		prevProc:  map[int]uint64{},
	}
}

func (s *Sampler) Sample() Usage {
	u := Usage{}
	if runtime.GOOS != "linux" {
		return u
	}
	now := time.Now()
	total, idle, cores := parseCPUStats(readFirst("/proc/stat"))
	if s.prevTotal > 0 && total > s.prevTotal {
		dt := total - s.prevTotal
		di := idle - s.prevIdle
		if dt > 0 {
			u.CPU = (1 - float64(di)/float64(dt)) * 100
			if u.CPU < 0 {
				u.CPU = 0
			}
		}
		for idx, c := range cores {
			if prev, ok := s.prevCores[idx]; ok && c.total > prev.total {
				dt := c.total - prev.total
				di := c.idle - prev.idle
				if dt > 0 {
					pct := (1 - float64(di)/float64(dt)) * 100
					if pct < 0 {
						pct = 0
					}
					u.Cores = append(u.Cores, pct)
				}
			}
		}
	}
	s.prevTotal, s.prevIdle = total, idle
	s.prevCores = cores
	sort.Sort(sort.Reverse(sort.Float64Slice(u.Cores)))
	if len(u.Cores) > maxCores {
		u.Cores = u.Cores[:maxCores]
	}

	memTotal, memAvail, swapTotal, swapFree := parseMemInfoFull(readFirst("/proc/meminfo"))
	u.MemTotal, u.MemAvail = memTotal, memAvail
	if swapTotal > swapFree {
		u.SwapTotal, u.SwapUsed = swapTotal, swapTotal-swapFree
	}
	u.Load1, u.Load5, u.Load15, u.ProcRunning, u.ProcTotal = parseLoadAvg(readFirst("/proc/loadavg"))
	u.DiskTotal, u.DiskUsed = diskUsage("/")

	net := parseNetDev(readFirst("/proc/net/dev"))
	if !s.prevNetTime.IsZero() && len(net) > 0 {
		if primary, ok := primaryIface(net); ok {
			if prev, ok := s.prevNet[primary]; ok {
				cur := net[primary]
				if cur.rx >= prev.rx && cur.tx >= prev.tx {
					elapsed := now.Sub(s.prevNetTime).Seconds()
					if elapsed > 0 {
						u.NetRx = float64(cur.rx-prev.rx) / elapsed
						u.NetTx = float64(cur.tx-prev.tx) / elapsed
					}
				}
			}
		}
	}
	s.prevNet = net
	s.prevNetTime = now

	u.Procs = s.topProcs(8, now)
	return u
}

// parseCPUStats parses /proc/stat aggregate plus per-core lines.
func parseCPUStats(content string) (total, idle uint64, cores map[int]coreStat) {
	cores = map[int]coreStat{}
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		name := fields[0]
		if name == "cpu" {
			total, idle = statTotals(fields[1:])
			continue
		}
		if strings.HasPrefix(name, "cpu") {
			if idx, err := strconv.Atoi(name[3:]); err == nil {
				t, i := statTotals(fields[1:])
				cores[idx] = coreStat{total: t, idle: i}
			}
		}
	}
	return total, idle, cores
}

func statTotals(fields []string) (total, idle uint64) {
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

// parseLoadAvg parses /proc/loadavg: "0.52 0.58 0.59 1/234 5678".
func parseLoadAvg(content string) (l1, l5, l15 float64, running, total int) {
	fields := strings.Fields(content)
	if len(fields) < 4 {
		return 0, 0, 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	if procs := strings.SplitN(fields[3], "/", 2); len(procs) == 2 {
		running, _ = strconv.Atoi(procs[0])
		total, _ = strconv.Atoi(procs[1])
	}
	return l1, l5, l15, running, total
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

// parseNetDev parses /proc/net/dev: "iface: rxbytes rxpkts ... txbytes ...".
func parseNetDev(content string) map[string]netStat {
	out := map[string]netStat{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") {
			continue
		}
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 9 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		out[strings.TrimSpace(name)] = netStat{rx: rx, tx: tx}
	}
	return out
}

func primaryIface(net map[string]netStat) (string, bool) {
	var best string
	var bestBytes uint64
	for name, st := range net {
		if name == "lo" {
			continue
		}
		if b := st.rx + st.tx; b > bestBytes {
			best, bestBytes = name, b
		}
	}
	return best, bestBytes > 0
}

func (s *Sampler) topProcs(n int, now time.Time) []Proc {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	page := uint64(os.Getpagesize())
	cores := runtime.NumCPU()
	elapsed := now.Sub(s.prevProcTime).Seconds()
	var procs []Proc
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		base := "/proc/" + e.Name()
		name := strings.TrimSpace(readFirst(base + "/comm"))
		if name == "" {
			continue
		}
		p := Proc{
			PID:  pid,
			Name: name,
			Mem:  parseStatmPages(readFirst(base+"/statm")) * page,
		}
		if ticks, ok := procTicks(readFirst(base + "/stat")); ok {
			if prev, had := s.prevProc[pid]; had && elapsed > 0 && ticks >= prev {
				p.CPU = float64(ticks-prev) / 100 / elapsed / float64(cores) * 100
			}
			s.prevProc[pid] = ticks
		}
		procs = append(procs, p)
	}
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPU != procs[j].CPU {
			return procs[i].CPU > procs[j].CPU
		}
		return procs[i].Mem > procs[j].Mem
	})
	var out []Proc
	for _, p := range procs {
		if p.Mem == 0 && p.CPU == 0 {
			continue // kernel threads / idle
		}
		out = append(out, p)
		if len(out) >= n {
			break
		}
	}
	s.prevProcTime = now
	return out
}

// procTicks extracts utime+stime (fields 14 and 15 of /proc/<pid>/stat)
// from the content after the comm field (which may contain parens).
func procTicks(content string) (uint64, bool) {
	idx := strings.LastIndexByte(content, ')')
	if idx < 0 || idx+2 > len(content) {
		return 0, false
	}
	fields := strings.Fields(content[idx+1:])
	// fields[0] is the state; utime and stime follow at offsets 11 and 12.
	if len(fields) < 13 {
		return 0, false
	}
	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return utime + stime, true
}

func parseStatmPages(content string) uint64 {
	fields := strings.Fields(content)
	if len(fields) < 2 {
		return 0
	}
	pages, _ := strconv.ParseUint(fields[1], 10, 64)
	return pages
}
