package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// cpuSample holds cumulative CPU tick counts from /proc/stat.
type cpuSample struct {
	user   int64
	system int64
	idle   int64
	ts     time.Time
}

// takeCPUSample reads /proc/stat and returns a cpuSample.
// Falls back to zero values on non-Linux systems.
func takeCPUSample() cpuSample {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSample{ts: time.Now()}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			break
		}
		parse := func(idx int) int64 {
			v, _ := strconv.ParseInt(fields[idx], 10, 64)
			return v
		}
		return cpuSample{
			user:   parse(1) + parse(2), // user + nice
			system: parse(3),
			idle:   parse(4),
			ts:     time.Now(),
		}
	}
	return cpuSample{ts: time.Now()}
}

// formatUptimeString converts seconds to a human-readable string.
func formatUptimeString(seconds float64) string {
	s := int64(seconds)
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, sec)
}
