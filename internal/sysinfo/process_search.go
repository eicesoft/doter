package sysinfo

import (
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// SearchProcesses matches every running process by name or executable path.
// It is a user-triggered query, so it may inspect the full process list; the
// regular snapshot only ships the union of the top CPU and memory rows.
func (s *Sampler) SearchProcesses(query string, limit int) []ProcessStats {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || limit <= 0 {
		return []ProcessStats{}
	}

	totalMem := s.Current().Memory.Total
	procs, err := process.Processes()
	if err != nil {
		return []ProcessStats{}
	}

	matches := make([]ProcessStats, 0, 16)
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		exe, _ := p.Exe()
		lowerName := strings.ToLower(name)
		lowerExe := strings.ToLower(exe)
		if !strings.Contains(lowerName, query) && !strings.Contains(lowerExe, query) {
			continue
		}

		stat := ProcessStats{PID: p.Pid, Name: name, Exe: exe}
		if created, err := p.CreateTime(); err == nil {
			stat.CreateTime = created
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			stat.MemoryRSS = mi.RSS
			if totalMem > 0 {
				stat.MemoryPercent = clamp(float64(mi.RSS)/float64(totalMem)*100, 0, 100)
			}
		}
		matches = append(matches, stat)
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].MemoryRSS == matches[j].MemoryRSS {
			return matches[i].Name < matches[j].Name
		}
		return matches[i].MemoryRSS > matches[j].MemoryRSS
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}
