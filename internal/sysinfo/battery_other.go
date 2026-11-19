//go:build !darwin || !cgo

package sysinfo

import "github.com/distatus/battery"

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
