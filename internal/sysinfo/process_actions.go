package sysinfo

import (
	"errors"
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v4/process"
)

// ErrProtectedProcess is returned when the UI asks to kill PID 1 or Doter
// itself.
var ErrProtectedProcess = errors.New("不能结束受保护的进程")

// KillProcess terminates a process after checking that its start time still
// matches the sampled one. The start-time check prevents a recycled PID from
// killing an unrelated process.
func KillProcess(pid int32, createTime int64) error {
	if pid <= 1 || pid == int32(os.Getpid()) {
		return ErrProtectedProcess
	}
	p, err := verifiedProcess(pid, createTime)
	if err != nil {
		return err
	}
	return p.Kill()
}

// RevealProcess opens the file manager at the process executable. The start
// time is checked for the same PID-reuse reason as KillProcess.
func RevealProcess(pid int32, createTime int64) bool {
	p, err := verifiedProcess(pid, createTime)
	if err != nil {
		return false
	}
	exe, err := p.Exe()
	if err != nil || exe == "" {
		return false
	}
	return revealFile(exe)
}

func verifiedProcess(pid int32, createTime int64) (*process.Process, error) {
	if createTime <= 0 {
		return nil, fmt.Errorf("无法读取进程启动时间")
	}
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}
	created, err := p.CreateTime()
	if err != nil {
		return nil, err
	}
	// Some platforms round start times differently; one second is enough to
	// reject a recycled PID without failing on clock granularity.
	if diff := created - createTime; diff < -1000 || diff > 1000 {
		return nil, fmt.Errorf("进程已发生变化，请刷新后重试")
	}
	return p, nil
}
