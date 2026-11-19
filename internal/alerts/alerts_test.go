package alerts

import (
	"testing"

	"doter/internal/agents"
	"doter/internal/sysinfo"
)

const t0 int64 = 1_700_000_000_000

func TestCPUAlertNeedsSustainedLoadAndHonoursCooldown(t *testing.T) {
	var got []Alert
	engine := New(func() bool { return true }, func(alert Alert) { got = append(got, alert) })

	engine.ObserveSnapshot(snapshotAt(t0, 95))
	engine.ObserveSnapshot(snapshotAt(t0+29_000, 95))
	if len(got) != 0 {
		t.Fatalf("alert fired before 30s: %+v", got)
	}

	engine.ObserveSnapshot(snapshotAt(t0+30_000, 95))
	if len(got) != 1 || got[0].Kind != KindCPU {
		t.Fatalf("expected one CPU alert, got %+v", got)
	}

	// Still high, but inside the cooldown.
	engine.ObserveSnapshot(snapshotAt(t0+60_000, 96))
	if len(got) != 1 {
		t.Fatalf("cooldown ignored: %+v", got)
	}

	// Falling below the clear threshold resets the hold timer.
	engine.ObserveSnapshot(snapshotAt(t0+16*60_000, 50))
	engine.ObserveSnapshot(snapshotAt(t0+16*60_000+10_000, 95))
	engine.ObserveSnapshot(snapshotAt(t0+16*60_000+40_000, 95))
	if len(got) != 2 {
		t.Fatalf("expected a second alert after recovery, got %+v", got)
	}
}

func TestDisabledEngineEmitsNothing(t *testing.T) {
	enabled := false
	var got []Alert
	engine := New(func() bool { return enabled }, func(alert Alert) { got = append(got, alert) })

	engine.ObserveSnapshot(snapshotAt(t0, 99))
	engine.ObserveSnapshot(snapshotAt(t0+60_000, 99))
	if len(got) != 0 {
		t.Fatalf("disabled engine emitted %+v", got)
	}
}

func TestLowDiskAndBatteryAlerts(t *testing.T) {
	var got []Alert
	engine := New(func() bool { return true }, func(alert Alert) { got = append(got, alert) })

	engine.ObserveSnapshot(sysinfo.Snapshot{
		Timestamp: t0,
		Disks: []sysinfo.DiskStats{{
			Mountpoint:  "/",
			Total:       100 << 30,
			Free:        4 << 30,
			UsedPercent: 96,
		}},
		Battery: &sysinfo.BatteryStats{Percent: 12, PluggedIn: false},
	})

	if len(got) != 2 {
		t.Fatalf("expected disk and battery alerts, got %+v", got)
	}
	if got[0].Kind != KindDisk || got[1].Kind != KindBattery {
		t.Fatalf("unexpected kinds: %+v", got)
	}
}

func TestAgentIdleAndDoneTransitions(t *testing.T) {
	var got []Alert
	engine := New(func() bool { return true }, func(alert Alert) { got = append(got, alert) })
	base := agents.Session{ID: "s1", Agent: "claude", ProjectName: "demo", Title: "Task"}

	running := base
	running.Status = agents.StatusRunning
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0, Sessions: []agents.Session{running}})

	// A quick turn is not worth interrupting the user.
	quick := base
	quick.Status = agents.StatusIdle
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 30_000, Sessions: []agents.Session{quick}})
	if len(got) != 0 {
		t.Fatalf("quick turn should not notify, got %+v", got)
	}

	// A long turn that ends waiting for input does notify.
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 70_000, Sessions: []agents.Session{running}})
	idle := base
	idle.Status = agents.StatusIdle
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 200_000, Sessions: []agents.Session{idle}})
	if len(got) != 1 || got[0].Kind != KindAgentIdle {
		t.Fatalf("expected idle alert for a long turn, got %+v", got)
	}

	// Repeating the same idle snapshot must not notify again.
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 260_000, Sessions: []agents.Session{idle}})
	if len(got) != 1 {
		t.Fatalf("duplicate idle alert: %+v", got)
	}

	// A new run that ends without a waiting turn produces a done alert.
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 270_000, Sessions: []agents.Session{running}})
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 280_000, Sessions: []agents.Session{running}})
	done := base
	done.Status = agents.StatusDone
	engine.ObserveAgents(agents.Snapshot{UpdatedAt: t0 + 290_000, Sessions: []agents.Session{done}})
	if len(got) != 2 || got[1].Kind != KindAgentDone {
		t.Fatalf("expected done alert, got %+v", got)
	}
}

func snapshotAt(ts int64, cpu float64) sysinfo.Snapshot {
	return sysinfo.Snapshot{
		Timestamp: ts,
		CPU:       sysinfo.CPUStats{Usage: cpu},
	}
}
