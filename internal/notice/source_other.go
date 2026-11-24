//go:build !darwin && !windows

package notice

import "errors"

var errUnsupported = errors.New("notification monitoring is not supported on this platform")

func platformFetch(string, int) ([]Entry, string, error) {
	return nil, "", errUnsupported
}

// OpenApp is a no-op where the notification store is unavailable.
func OpenApp(string) error { return errUnsupported }
