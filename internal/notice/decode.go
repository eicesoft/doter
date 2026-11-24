package notice

import (
	"encoding/xml"
	"strings"

	"howett.net/plist"
)

// appleEpoch is the offset in seconds between Unix time and Apple's
// 2001-01-01 epoch, which is what the macOS notification store records.
const appleEpoch = 978307200

func appleToUnixMilli(seconds float64) int64 {
	if seconds <= 0 {
		return 0
	}
	return int64((seconds + appleEpoch) * 1000)
}

// darwinPayload is the binary plist stored in a record's data column.
type darwinPayload struct {
	App  string  `plist:"app"`
	Date float64 `plist:"date"`
	Req  struct {
		Title string `plist:"titl"`
		Body  string `plist:"body"`
	} `plist:"req"`
}

func decodeDarwin(data []byte) (darwinPayload, bool) {
	var payload darwinPayload
	if len(data) == 0 {
		return payload, false
	}
	if _, err := plist.Unmarshal(data, &payload); err != nil {
		return payload, false
	}
	return payload, true
}

// filetimeEpoch is the number of 100ns ticks between 1601-01-01 and
// 1970-01-01.
const filetimeEpoch = 116444736000000000

func filetimeToUnixMilli(ticks int64) int64 {
	if ticks <= filetimeEpoch {
		return 0
	}
	return (ticks - filetimeEpoch) / 10000
}

// toastPayload is the XML document Windows stores in a notification's payload
// column. Every visible line is a <text> element under visual/binding.
type toastPayload struct {
	Visual struct {
		Binding struct {
			Texts []string `xml:"text"`
		} `xml:"binding"`
	} `xml:"visual"`
}

func parseToastPayload(raw []byte) (title, body string) {
	var toast toastPayload
	if err := xml.Unmarshal(raw, &toast); err != nil {
		return "", ""
	}
	texts := toast.Visual.Binding.Texts
	if len(texts) == 0 {
		return "", ""
	}
	return texts[0], strings.Join(texts[1:], " ")
}
