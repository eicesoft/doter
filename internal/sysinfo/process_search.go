package sysinfo

import (
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// SearchProcesses matches every running process by name or executable path.
// It is a user-triggered query, so it may inspect the full process list; the
// regular snapshot only ships the union of the top CPU and memory rows.
//
// Every hit carries a CPU% reading: the delta against the sampler's previous
// sample when the process was already known, otherwise its lifetime average.
// The result is the union of the strongest CPU and memory matches, so ranking
// by either column in the UI still sees the relevant processes.
func (s *Sampler) SearchProcesses(query string, limit int) []ProcessStats {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || limit <= 0 {
		return []ProcessStats{}
	}

	totalMem := s.Current().Memory.Total
	prev := s.procBaseline()
	now := time.Now()

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
		if times, err := p.Times(); err == nil {
			stat.CPUPercent = processCPUPercent(prev, now, p.Pid, times.User+times.System, stat.CreateTime)
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			stat.MemoryRSS = mi.RSS
			if totalMem > 0 {
				stat.MemoryPercent = clamp(float64(mi.RSS)/float64(totalMem)*100, 0, 100)
			}
		}
		matches = append(matches, stat)
	}

	return mergeProcessRanking(matches, limit)
}

// procBaseline copies the CPU counters captured by the latest sample so a
// search can diff against them without racing the sampler.
func (s *Sampler) procBaseline() map[int32]procCounters {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prev := make(map[int32]procCounters, len(s.lastProc))
	for pid, counters := range s.lastProc {
		prev[pid] = counters
	}
	return prev
}

// processCPUPercent prefers the recent delta against the last sample. A
// process that was not part of that sample (started since, or missed) falls
// back to its lifetime average, which is coarse but never blank.
func processCPUPercent(prev map[int32]procCounters, now time.Time, pid int32, cpuTime float64, createTime int64) float64 {
	if old, ok := prev[pid]; ok {
		if elapsed := now.Sub(old.at).Seconds(); elapsed > 0 && cpuTime >= old.cpuTime {
			return clamp((cpuTime-old.cpuTime)/elapsed*100, 0, 100)
		}
	}
	if createTime > 0 {
		if elapsed := now.Sub(time.UnixMilli(createTime)).Seconds(); elapsed > 0 {
			return clamp(cpuTime/elapsed*100, 0, 100)
		}
	}
	return 0
}

// mergeProcessRanking keeps the top half of both the CPU and memory rankings
// so the caller can sort by either column without a quiet-but-heavy process
// being cut before the UI ever sees it.
func mergeProcessRanking(matches []ProcessStats, limit int) []ProcessStats {
	byCPU := func(a, b ProcessStats) bool {
		if a.CPUPercent == b.CPUPercent {
			return a.MemoryRSS > b.MemoryRSS
		}
		return a.CPUPercent > b.CPUPercent
	}

	if len(matches) <= limit {
		sort.Slice(matches, func(i, j int) bool { return byCPU(matches[i], matches[j]) })
		return matches
	}

	byMem := append([]ProcessStats(nil), matches...)
	sort.Slice(byMem, func(i, j int) bool {
		if byMem[i].MemoryRSS == byMem[j].MemoryRSS {
			return byMem[i].CPUPercent > byMem[j].CPUPercent
		}
		return byMem[i].MemoryRSS > byMem[j].MemoryRSS
	})
	sortedCPU := append([]ProcessStats(nil), matches...)
	sort.Slice(sortedCPU, func(i, j int) bool { return byCPU(sortedCPU[i], sortedCPU[j]) })

	per := limit / 2
	if per < 1 {
		per = 1
	}
	if len(byMem) > per {
		byMem = byMem[:per]
	}
	if len(sortedCPU) > per {
		sortedCPU = sortedCPU[:per]
	}

	merged := make([]ProcessStats, 0, 2*per)
	seen := make(map[int32]struct{}, 2*per)
	for _, list := range [][]ProcessStats{sortedCPU, byMem} {
		for _, p := range list {
			if _, ok := seen[p.PID]; ok {
				continue
			}
			seen[p.PID] = struct{}{}
			merged = append(merged, p)
		}
	}
	sort.Slice(merged, func(i, j int) bool { return byCPU(merged[i], merged[j]) })
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}
