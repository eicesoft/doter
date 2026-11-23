package sysinfo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearchProcessesFindsCurrentProcess(t *testing.T) {
	query := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	sampler := NewSampler()
	results := sampler.SearchProcesses(query, 20)
	if len(results) == 0 {
		t.Fatalf("no process matched %q", query)
	}
	for _, result := range results {
		if result.PID == int32(os.Getpid()) {
			return
		}
	}
	t.Fatalf("current process not in results: %+v", results)
}

func TestKillProcessProtectsSelfAndInit(t *testing.T) {
	if err := KillProcess(int32(os.Getpid()), 0); !errors.Is(err, ErrProtectedProcess) {
		t.Fatalf("self kill = %v, want ErrProtectedProcess", err)
	}
	if err := KillProcess(1, 0); !errors.Is(err, ErrProtectedProcess) {
		t.Fatalf("PID 1 kill = %v, want ErrProtectedProcess", err)
	}
}

func TestVerifiedProcessRequiresStartTime(t *testing.T) {
	if _, err := verifiedProcess(int32(os.Getpid()), 0); err == nil {
		t.Fatal("missing start time should be rejected")
	}
}

func TestProcessCPUPercentPrefersRecentDelta(t *testing.T) {
	now := time.Now()
	createTime := now.Add(-10 * time.Second).UnixMilli()

	// Without a previous sample the lifetime average is the only reading.
	if got := processCPUPercent(nil, now, 42, 1, createTime); !almostEqual(got, 10) {
		t.Fatalf("lifetime average = %v, want 10", got)
	}

	// A sample from two seconds ago with 0.5s of CPU time yields 25%.
	prev := map[int32]procCounters{42: {cpuTime: 0.5, at: now.Add(-2 * time.Second)}}
	if got := processCPUPercent(prev, now, 42, 1, createTime); !almostEqual(got, 25) {
		t.Fatalf("delta = %v, want 25", got)
	}
}

func almostEqual(got, want float64) bool {
	const epsilon = 0.001
	diff := got - want
	return diff < epsilon && diff > -epsilon
}

func TestMergeProcessRankingKeepsTopOfBothRankings(t *testing.T) {
	matches := []ProcessStats{
		{PID: 1, Name: "hot", CPUPercent: 90, MemoryRSS: 1},
		{PID: 2, Name: "heavy", CPUPercent: 1, MemoryRSS: 1 << 30},
	}
	for pid := int32(3); pid <= 8; pid++ {
		matches = append(matches, ProcessStats{PID: pid, CPUPercent: 5, MemoryRSS: 1 << 20})
	}

	merged := mergeProcessRanking(matches, 4)
	if len(merged) > 4 {
		t.Fatalf("merged %d rows, want at most 4", len(merged))
	}
	for _, want := range []string{"hot", "heavy"} {
		found := false
		for _, row := range merged {
			if row.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s dropped from %+v", want, merged)
		}
	}
}
