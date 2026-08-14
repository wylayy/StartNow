package machine

import (
	"runtime"
	"testing"
)

func TestParseCPUStats(t *testing.T) {
	content := `cpu  100 20 30 400 50 10 5 0 0 0
cpu0 10 2 3 40 5 1 0 0 0 0
cpu1 20 4 6 80 10 2 0 0 0 0
`
	total, idle, cores := parseCPUStats(content)
	if total != 100+20+30+400+50+10+5 {
		t.Errorf("total = %d", total)
	}
	if idle != 400+50 {
		t.Errorf("idle = %d", idle)
	}
	if len(cores) != 2 {
		t.Fatalf("cores = %d", len(cores))
	}
	if cores[1].total != 20+4+6+80+10+2 {
		t.Errorf("core1 total = %d", cores[1].total)
	}
}

func TestParseMemInfoFull(t *testing.T) {
	content := "MemTotal:       32768000 kB\nMemAvailable:   19000000 kB\nSwapTotal:       4194304 kB\nSwapFree:        1048576 kB\n"
	total, avail, swapTotal, swapFree := parseMemInfoFull(content)
	if total != 32768000*1024 || avail != 19000000*1024 {
		t.Errorf("mem: %d %d", total, avail)
	}
	if swapTotal != 4194304*1024 || swapFree != 1048576*1024 {
		t.Errorf("swap: %d %d", swapTotal, swapFree)
	}
}

func TestParseLoadAvg(t *testing.T) {
	l1, l5, l15, running, total := parseLoadAvg("0.52 0.58 0.59 1/234 5678")
	if l1 != 0.52 || l5 != 0.58 || l15 != 0.59 {
		t.Errorf("got %v %v %v", l1, l5, l15)
	}
	if running != 1 || total != 234 {
		t.Errorf("procs: %d/%d", running, total)
	}
	if l1, _, _, _, _ = parseLoadAvg("garbage"); l1 != 0 {
		t.Error("expected 0 for garbage")
	}
}

func TestParseNetDev(t *testing.T) {
	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000     10    0    0    0     0          0         0     1000     10    0    0    0     0       0          0
  eth0: 2000     20    0    0    0     0          0         0     3000     30    0    0    0     0       0          0
`
	net := parseNetDev(content)
	if len(net) != 2 {
		t.Fatalf("interfaces = %d", len(net))
	}
	if net["eth0"].rx != 2000 || net["eth0"].tx != 3000 {
		t.Errorf("eth0: %+v", net["eth0"])
	}
	primary, ok := primaryIface(net)
	if !ok || primary != "eth0" {
		t.Errorf("primary = %q ok=%v", primary, ok)
	}
}

func TestProcTicks(t *testing.T) {
	content := "1234 (bash) S 1 1234 1234 0 -1 4194560 123 0 0 0 30 40 20 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	ticks, ok := procTicks(content)
	if !ok || ticks != 70 {
		t.Errorf("ticks = %d ok=%v, want 70", ticks, ok)
	}
	if _, ok := procTicks("garbage"); ok {
		t.Error("expected failure for garbage")
	}
}

func TestParseStatmPages(t *testing.T) {
	if got := parseStatmPages("100 25 10 5 0 0 0"); got != 25 {
		t.Errorf("got %d", got)
	}
	if got := parseStatmPages(""); got != 0 {
		t.Errorf("empty: got %d", got)
	}
}

func TestSamplerDelta(t *testing.T) {
	s := NewSampler()
	_ = s.Sample()
	if runtime.GOOS == "linux" {
		u := s.Sample()
		if u.CPU < 0 || u.CPU > 100 {
			t.Errorf("cpu out of range: %v", u.CPU)
		}
		if u.MemTotal == 0 {
			t.Error("mem total empty on linux")
		}
		if u.DiskTotal == 0 {
			t.Error("disk total empty on linux")
		}
		if u.ProcTotal == 0 {
			t.Error("proc total empty on linux")
		}
		if len(u.Cores) == 0 {
			t.Error("per-core stats empty on linux")
		}
	}
}
