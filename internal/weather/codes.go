package weather

// descriptions maps WMO weather codes, as used by Open-Meteo, to Chinese
// labels. Codes are grouped: 0 clear, 1–3 cloud cover, 45–48 fog, 51–57
// drizzle, 61–67 rain, 71–77 snow, 80–86 showers, 95–99 thunderstorms.
var descriptions = map[int]string{
	0:  "晴",
	1:  "晴间多云",
	2:  "多云",
	3:  "阴",
	45: "雾",
	48: "雾凇",
	51: "小毛毛雨",
	53: "毛毛雨",
	55: "大毛毛雨",
	56: "冻毛毛雨",
	57: "强冻毛毛雨",
	61: "小雨",
	63: "中雨",
	65: "大雨",
	66: "冻雨",
	67: "强冻雨",
	71: "小雪",
	73: "中雪",
	75: "大雪",
	77: "雪粒",
	80: "小阵雨",
	81: "阵雨",
	82: "强阵雨",
	85: "小阵雪",
	86: "强阵雪",
	95: "雷阵雨",
	96: "雷阵雨伴冰雹",
	99: "强雷暴伴冰雹",
}

// Describe returns the Chinese label for a WMO weather code.
func Describe(code int) string {
	if d, ok := descriptions[code]; ok {
		return d
	}
	return "未知"
}
