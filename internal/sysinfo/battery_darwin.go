//go:build darwin && cgo

package sysinfo

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <IOKit/ps/IOPowerSources.h>
#include <IOKit/ps/IOPSKeys.h>

// readBattery copies the machine's battery state into the out params and
// returns 0 when there is no battery (e.g. a desktop Mac).
static int readBattery(double *percent, int *charging, int *onAC, int *secondsLeft) {
	CFTypeRef blob = IOPSCopyPowerSourcesInfo();
	if (blob == NULL) {
		return 0;
	}
	CFArrayRef sources = IOPSCopyPowerSourcesList(blob);
	if (sources == NULL) {
		CFRelease(blob);
		return 0;
	}

	int found = 0;
	for (CFIndex i = 0; i < CFArrayGetCount(sources) && !found; i++) {
		CFDictionaryRef desc = IOPSGetPowerSourceDescription(blob, CFArrayGetValueAtIndex(sources, i));
		if (desc == NULL) {
			continue;
		}

		int current = 0, max = 0, isCharging = 0, minutes = -1;
		CFNumberRef n;
		if ((n = CFDictionaryGetValue(desc, CFSTR(kIOPSCurrentCapacityKey))) != NULL) {
			CFNumberGetValue(n, kCFNumberIntType, &current);
		}
		if ((n = CFDictionaryGetValue(desc, CFSTR(kIOPSMaxCapacityKey))) != NULL) {
			CFNumberGetValue(n, kCFNumberIntType, &max);
		}
		if (max <= 0) {
			continue;
		}
		if ((n = CFDictionaryGetValue(desc, CFSTR(kIOPSIsChargingKey))) != NULL) {
			CFNumberGetValue(n, kCFNumberIntType, &isCharging);
		}

		CFStringRef sourceState = CFDictionaryGetValue(desc, CFSTR(kIOPSPowerSourceStateKey));
		int ac = sourceState != NULL && CFStringCompare(sourceState, CFSTR(kIOPSACPowerValue), 0) == kCFCompareEqualTo;
		if (!ac && (n = CFDictionaryGetValue(desc, CFSTR(kIOPSTimeToEmptyKey))) != NULL) {
			CFNumberGetValue(n, kCFNumberIntType, &minutes);
		}

		*percent = (double)current / (double)max * 100.0;
		*charging = isCharging;
		*onAC = ac;
		*secondsLeft = minutes > 0 ? minutes * 60 : -1;
		found = 1;
	}

	CFRelease(sources);
	CFRelease(blob);
	return found;
}
*/
import "C"

// collectBattery reads the battery through IOKit's power source API, which
// keeps working on Apple silicon where SMC-key readers (distatus/battery)
// report an empty battery.
func collectBattery() *BatteryStats {
	var percent C.double
	var charging, onAC, secondsLeft C.int
	if C.readBattery(&percent, &charging, &onAC, &secondsLeft) == 0 {
		return nil
	}

	state := "Discharging"
	switch {
	case onAC != 0 && charging != 0:
		state = "Charging"
	case onAC != 0 && percent >= 99.5:
		state = "Full"
	case onAC != 0:
		state = "Idle"
	}

	return &BatteryStats{
		Percent:     clamp(float64(percent), 0, 100),
		State:       state,
		PluggedIn:   onAC != 0,
		SecondsLeft: int64(secondsLeft),
	}
}
