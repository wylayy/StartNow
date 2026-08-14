package machine

import (
	"testing"
	"time"
)

func TestParseCPUInfo(t *testing.T) {
	content := `processor	: 0
vendor_id	: GenuineIntel
model name	: AMD Ryzen 7 5800X 8-Core Processor
processor	: 1
model name	: AMD Ryzen 7 5800X 8-Core Processor
`
	model, cpus := parseCPUInfo(content)
	if cpus != 2 {
		t.Errorf("cpus = %d, want 2", cpus)
	}
	if model != "AMD Ryzen 7 5800X 8-Core Processor" {
		t.Errorf("model = %q", model)
	}
}

func TestParseMemInfo(t *testing.T) {
	content := "MemTotal:       32768000 kB\nMemFree:        1000000 kB\nMemAvailable:   19000000 kB\n"
	total, avail := parseMemInfo(content)
	if total != 32768000*1024 {
		t.Errorf("total = %d", total)
	}
	if avail != 19000000*1024 {
		t.Errorf("avail = %d", avail)
	}
}

func TestParseUptime(t *testing.T) {
	d := parseUptime("123456.78 987654.32")
	want := time.Duration(123456.78 * float64(time.Second))
	if d != want {
		t.Errorf("got %v, want %v", d, want)
	}
	if parseUptime("garbage") != 0 {
		t.Error("expected 0 for garbage")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		n    uint64
		want string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{3 * 1024 * 1024 * 1024, "3.0 GB"},
	}
	for _, c := range cases {
		if got := HumanBytes(c.n); got != c.want {
			t.Errorf("HumanBytes(%d) = %s, want %s", c.n, got, c.want)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Minute, "45m"},
		{3*time.Hour + 12*time.Minute, "3h 12m"},
		{2*24*time.Hour + 4*time.Hour + 5*time.Minute, "2d 4h 5m"},
	}
	for _, c := range cases {
		if got := HumanDuration(c.d); got != c.want {
			t.Errorf("HumanDuration(%v) = %s, want %s", c.d, got, c.want)
		}
	}
}
