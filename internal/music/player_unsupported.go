//go:build (windows && !cgo) || (!windows && !darwin)

package music

import (
	"context"
	"errors"
)

type unsupportedPlayer struct {
	reason string
}

func newPlayer() Player {
	return unsupportedPlayer{reason: "此平台暂不支持读取系统媒体"}
}

func (p unsupportedPlayer) Start(context.Context) error {
	return errors.New(p.reason)
}

func (p unsupportedPlayer) Snapshot() NowPlaying {
	return NowPlaying{Error: p.reason}
}

func (p unsupportedPlayer) Artwork(string) string { return "" }

func (p unsupportedPlayer) PlayPause() error { return errors.New(p.reason) }
func (p unsupportedPlayer) Next() error      { return errors.New(p.reason) }
func (p unsupportedPlayer) Previous() error  { return errors.New(p.reason) }
func (p unsupportedPlayer) Seek(float64) error {
	return errors.New(p.reason)
}
func (p unsupportedPlayer) ToggleShuffle() error { return errors.New(p.reason) }
func (p unsupportedPlayer) CycleRepeat() error   { return errors.New(p.reason) }
func (p unsupportedPlayer) Close()               {}
