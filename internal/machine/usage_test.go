package machine

import (
	"runtime"
	"testing"
)

func TestParseCPUStat(t *testing.T) {
	content := `cpu  100 20 30 400 50 10 5 0 0 0
cpu0 10 2 3 40 5 1 0 0 0 0
`
	total, idle := parseCPUStat(content)
	if total != 100+20+30+400+50+10+5 {
		t.Errorf("total = %d", total)
	}
	if idle != 400+50 {
		t.Errorf("idle = %d", idle)
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
	l1, l5, l15 := parseLoadAvg("0.52 0.58 0.59 1/234 5678")
	if l1 != 0.52 || l5 != 0.58 || l15 != 0.59 {
		t.Errorf("got %v %v %v", l1, l5, l15)
	}
	if l1, _, _ = parseLoadAvg("garbage"); l1 != 0 {
		t.Error("expected 0 for garbage")
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
	}
}
