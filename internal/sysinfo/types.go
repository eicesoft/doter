// Package sysinfo collects cross-platform system metrics for the notch UI.
package sysinfo

// StaticInfo describes hardware and operating system details that do not
// change while the app is running. It is fetched once and exposed to the
// frontend so the panel can show the machine's identity.
type StaticInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	KernelArch      string `json:"kernelArch"`
	KernelVersion   string `json:"kernelVersion"`
	BootTime        int64  `json:"bootTime"`
	CPUModel        string `json:"cpuModel"`
	CPUCores        int    `json:"cpuCores"`
	CPUThreads      int    `json:"cpuThreads"`
	MemoryTotal     uint64 `json:"memoryTotal"`
}

// Snapshot is a single point-in-time reading of the machine.
type Snapshot struct {
	Timestamp int64 `json:"timestamp"`
	Uptime    int64 `json:"uptime"`

	CPU       CPUStats       `json:"cpu"`
	Memory    MemoryStats    `json:"memory"`
	Swap      MemoryStats    `json:"swap"`
	Disks     []DiskStats    `json:"disks"`
	Network   NetworkStats   `json:"network"`
	Battery   *BatteryStats  `json:"battery,omitempty"`
	Load      *LoadStats     `json:"load,omitempty"`
	Processes []ProcessStats `json:"processes"`
}

type CPUStats struct {
	Usage   float64   `json:"usage"`
	PerCore []float64 `json:"perCore"`
}

type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"usedPercent"`
}

type DiskStats struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
}

type NetworkStats struct {
	UpSpeed    float64        `json:"upSpeed"`
	DownSpeed  float64        `json:"downSpeed"`
	TotalSent  uint64         `json:"totalSent"`
	TotalRecv  uint64         `json:"totalRecv"`
	Interfaces []InterfaceNet `json:"interfaces"`
}

type InterfaceNet struct {
	Name      string  `json:"name"`
	UpSpeed   float64 `json:"upSpeed"`
	DownSpeed float64 `json:"downSpeed"`
	Sent      uint64  `json:"sent"`
	Recv      uint64  `json:"recv"`
}

type BatteryStats struct {
	Percent     float64 `json:"percent"`
	State       string  `json:"state"`
	PluggedIn   bool    `json:"pluggedIn"`
	SecondsLeft int64   `json:"secondsLeft"` // -1 when unknown
}

type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type ProcessStats struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	MemoryRSS     uint64  `json:"memoryRSS"`
}
