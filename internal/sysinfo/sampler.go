package sysinfo

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Sampler periodically reads system metrics on a single background goroutine.
// Consumers read the latest snapshot through Current or receive copies on the
// Updates channel.
type Sampler struct {
	mu       sync.RWMutex
	snapshot Snapshot
	static   StaticInfo

	updates chan Snapshot
	started bool

	tick       int
	bootTime   uint64
	lastSample time.Time
	lastNet    map[string]netCounters
	lastProc   map[int32]procCounters
}

type netCounters struct {
	sent uint64
	recv uint64
}

type procCounters struct {
	cpuTime float64
	at      time.Time
}

// NewSampler builds a sampler and captures the static machine information.
func NewSampler() *Sampler {
	s := &Sampler{
		updates:  make(chan Snapshot, 1),
		lastNet:  make(map[string]netCounters),
		lastProc: make(map[int32]procCounters),
	}
	s.static = collectStatic()
	s.bootTime, _ = host.BootTime()
	s.snapshot = Snapshot{
		Timestamp: time.Now().UnixMilli(),
		CPU:       CPUStats{PerCore: []float64{}},
		Disks:     []DiskStats{},
		Processes: []ProcessStats{},
	}
	return s
}

// Start begins sampling once per second until ctx is cancelled.
func (s *Sampler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	go func() {
		s.sampleOnce()
		s.publish()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sampleOnce()
				s.publish()
			}
		}
	}()
}

func (s *Sampler) publish() {
	snap := s.Current()
	select {
	case s.updates <- snap:
	default:
		// The consumer is behind; drop this frame rather than queueing.
	}
}

// Updates yields a snapshot after every sample. The channel is never closed.
func (s *Sampler) Updates() <-chan Snapshot { return s.updates }

// Current returns the most recent snapshot.
func (s *Sampler) Current() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

// Static returns immutable machine details captured at startup.
func (s *Sampler) Static() StaticInfo { return s.static }

func (s *Sampler) sampleOnce() {
	now := time.Now()

	s.mu.RLock()
	snap := s.snapshot
	lastSample := s.lastSample
	prevNet := s.lastNet
	prevProc := s.lastProc
	tick := s.tick
	s.mu.RUnlock()

	elapsed := now.Sub(lastSample).Seconds()
	if elapsed <= 0 || elapsed > 10 {
		elapsed = 1
	}

	if s.bootTime > 0 {
		snap.Uptime = int64(now.Unix()) - int64(s.bootTime)
	}
	snap.Timestamp = now.UnixMilli()

	// CPU: percentage since the previous call in this goroutine.
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		snap.CPU.Usage = clamp(pct[0], 0, 100)
	}
	if perCore, err := cpu.Percent(0, true); err == nil && len(perCore) > 0 {
		for i := range perCore {
			perCore[i] = clamp(perCore[i], 0, 100)
		}
		snap.CPU.PerCore = perCore
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		snap.Memory = MemoryStats{
			Total:       vm.Total,
			Used:        vm.Used,
			Available:   vm.Available,
			UsedPercent: clamp(vm.UsedPercent, 0, 100),
		}
	}
	if sw, err := mem.SwapMemory(); err == nil && sw.Total > 0 {
		snap.Swap = MemoryStats{
			Total:       sw.Total,
			Used:        sw.Used,
			Available:   sw.Free,
			UsedPercent: clamp(sw.UsedPercent, 0, 100),
		}
	} else {
		snap.Swap = MemoryStats{}
	}

	if avg, err := load.Avg(); err == nil {
		snap.Load = &LoadStats{Load1: avg.Load1, Load5: avg.Load5, Load15: avg.Load15}
	}

	netStats, counters := collectNetwork(prevNet, elapsed)
	snap.Network = netStats

	// Heavier collectors run less often to keep the one-second cadence cheap.
	if tick%5 == 0 {
		snap.Disks = collectDisks()
		var procCountersNow map[int32]procCounters
		snap.Processes, procCountersNow = collectProcesses(prevProc, now, snap.Memory.Total)
		prevProc = procCountersNow
	}
	if tick%10 == 0 {
		snap.Battery = collectBattery()
	}

	s.mu.Lock()
	s.tick++
	s.lastSample = now
	s.lastNet = counters
	s.lastProc = prevProc
	s.snapshot = snap
	s.mu.Unlock()
}

func collectStatic() StaticInfo {
	info := StaticInfo{}
	if h, err := host.Info(); err == nil {
		info.Hostname = h.Hostname
		info.OS = h.OS
		info.Platform = h.Platform
		info.PlatformVersion = h.PlatformVersion
		info.KernelArch = h.KernelArch
		info.KernelVersion = h.KernelVersion
		info.BootTime = int64(h.BootTime)
	}
	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		info.CPUModel = strings.TrimSpace(infos[0].ModelName)
	}
	if n, err := cpu.Counts(false); err == nil {
		info.CPUCores = n
	}
	if n, err := cpu.Counts(true); err == nil {
		info.CPUThreads = n
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		info.MemoryTotal = vm.Total
	}
	return info
}

func collectNetwork(prev map[string]netCounters, elapsed float64) (NetworkStats, map[string]netCounters) {
	counters, err := gopsnet.IOCounters(true)
	if err != nil {
		return NetworkStats{Interfaces: []InterfaceNet{}}, map[string]netCounters{}
	}

	stats := NetworkStats{Interfaces: make([]InterfaceNet, 0, len(counters))}
	next := make(map[string]netCounters, len(counters))

	for _, c := range counters {
		if isLoopback(c.Name) {
			continue
		}
		name := c.Name
		if name == "" {
			name = "default"
		}
		next[name] = netCounters{sent: c.BytesSent, recv: c.BytesRecv}

		iface := InterfaceNet{
			Name: name,
			Sent: c.BytesSent,
			Recv: c.BytesRecv,
		}
		if old, ok := prev[name]; ok {
			if c.BytesSent >= old.sent {
				iface.UpSpeed = float64(c.BytesSent-old.sent) / elapsed
			}
			if c.BytesRecv >= old.recv {
				iface.DownSpeed = float64(c.BytesRecv-old.recv) / elapsed
			}
		}
		iface.UpSpeed = mathMax0(iface.UpSpeed)
		iface.DownSpeed = mathMax0(iface.DownSpeed)

		stats.UpSpeed += iface.UpSpeed
		stats.DownSpeed += iface.DownSpeed
		stats.TotalSent += c.BytesSent
		stats.TotalRecv += c.BytesRecv

		if iface.Sent+iface.Recv > 0 {
			stats.Interfaces = append(stats.Interfaces, iface)
		}
	}

	sort.Slice(stats.Interfaces, func(i, j int) bool {
		a, b := stats.Interfaces[i], stats.Interfaces[j]
		return (a.UpSpeed + a.DownSpeed) > (b.UpSpeed + b.DownSpeed)
	})
	if len(stats.Interfaces) > 4 {
		stats.Interfaces = stats.Interfaces[:4]
	}
	return stats, next
}

func collectDisks() []DiskStats {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return []DiskStats{}
	}

	seen := make(map[string]struct{}, len(partitions))
	result := make([]DiskStats, 0, len(partitions))
	for _, p := range partitions {
		if p.Mountpoint == "" || p.Device == "" {
			continue
		}
		if skipFilesystem(p.Fstype, p.Mountpoint) {
			continue
		}
		if _, ok := seen[p.Mountpoint]; ok {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		seen[p.Mountpoint] = struct{}{}
		result = append(result, DiskStats{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: clamp(usage.UsedPercent, 0, 100),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Mountpoint < result[j].Mountpoint
	})
	if len(result) > 4 {
		result = result[:4]
	}
	return result
}

func skipFilesystem(fstype, mountpoint string) bool {
	switch fstype {
	case "devfs", "devtmpfs", "tmpfs", "overlay", "squashfs", "aufs", "proc", "sysfs", "cgroup", "cgroup2", "none":
		return true
	}
	if strings.HasPrefix(mountpoint, "/snap") || strings.HasPrefix(mountpoint, "/var/snap") {
		return true
	}
	return false
}

func collectProcesses(prev map[int32]procCounters, now time.Time, totalMem uint64) ([]ProcessStats, map[int32]procCounters) {
	procs, err := process.Processes()
	if err != nil {
		return []ProcessStats{}, prev
	}

	next := make(map[int32]procCounters, len(procs))
	result := make([]ProcessStats, 0, len(procs))
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		stat := ProcessStats{PID: p.Pid, Name: name}

		if times, err := p.Times(); err == nil {
			total := times.User + times.System
			next[p.Pid] = procCounters{cpuTime: total, at: now}
			if old, ok := prev[p.Pid]; ok {
				delta := now.Sub(old.at).Seconds()
				if delta > 0 && total >= old.cpuTime {
					stat.CPUPercent = clamp((total-old.cpuTime)/delta*100, 0, 100)
				}
			}
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			stat.MemoryRSS = mi.RSS
			if totalMem > 0 {
				stat.MemoryPercent = clamp(float64(mi.RSS)/float64(totalMem)*100, 0, 100)
			}
		}
		result = append(result, stat)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].CPUPercent == result[j].CPUPercent {
			return result[i].MemoryRSS > result[j].MemoryRSS
		}
		return result[i].CPUPercent > result[j].CPUPercent
	})
	if len(result) > 5 {
		result = result[:5]
	}
	return result, next
}

func collectBattery() *BatteryStats {
	b, err := battery.Get(0)
	if err != nil || b == nil || b.Full <= 0 {
		return nil
	}

	percent := clamp(b.Current/b.Full*100, 0, 100)
	state := b.State.String()
	pluggedIn := b.State.Raw == battery.Charging || b.State.Raw == battery.Full || b.State.Raw == battery.Idle

	secondsLeft := int64(-1)
	if b.State.Raw == battery.Discharging && b.ChargeRate > 0 {
		secondsLeft = int64(b.Current / b.ChargeRate * 3600)
	}

	return &BatteryStats{
		Percent:     percent,
		State:       state,
		PluggedIn:   pluggedIn,
		SecondsLeft: secondsLeft,
	}
}

func isLoopback(name string) bool {
	lower := strings.ToLower(name)
	return lower == "lo" || lower == "lo0" || strings.Contains(lower, "loopback")
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func mathMax0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
