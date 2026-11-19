package sysinfo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
