//go:build (!darwin || !cgo) && !windows && !linux

package clipboard

import "errors"

func rememberTarget() {}

func pasteTarget() error {
	return errors.New("当前平台不支持自动粘贴")
}
