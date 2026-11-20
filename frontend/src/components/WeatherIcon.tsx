/** WMO weather code groups the icon set is drawn from. */
type Kind = "clear" | "partly" | "cloudy" | "fog" | "rain" | "snow" | "storm";

function kindOf(code: number): Kind {
  switch (code) {
    case 0:
      return "clear";
    case 1:
    case 2:
      return "partly";
    case 3:
      return "cloudy";
    case 45:
    case 48:
      return "fog";
    case 51:
    case 53:
    case 55:
    case 56:
    case 57:
    case 61:
    case 63:
    case 65:
    case 66:
    case 67:
    case 80:
    case 81:
    case 82:
      return "rain";
    case 71:
    case 73:
    case 75:
    case 77:
    case 85:
    case 86:
      return "snow";
    default:
      return code >= 95 ? "storm" : "cloudy";
  }
}

/** The tab's key colour, picked from the current glyph family: the sky the
    tab is reporting sets the accent for the card, hero, metrics and tab bar. */
export function weatherAccent(code: number, night = false): string {
  switch (kindOf(code)) {
    case "clear":
      return night ? "#a9b6ff" : "#fbbf24";
    case "partly":
      return night ? "#93b4ff" : "#7dd3fc";
    case "cloudy":
      return "#a9b4c4";
    case "fog":
      return "#8f9bab";
    case "rain":
      return "#38bdf8";
    case "snow":
      return "#d8f0ff";
    case "storm":
      return "#c084fc";
  }
}

/** A minimal weather glyph drawn in the panel's stroke style. `night` swaps
    the sun for a moon on the clear and partly-cloudy glyphs. Small sizes get
    a heavier stroke so the glyph stays legible in the idle pill. */
export function WeatherIcon({
  code,
  size = 20,
  night = false,
}: {
  code: number;
  size?: number;
  night?: boolean;
}) {
  const kind = kindOf(code);
  const props = {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: size < 14 ? 2.2 : 1.7,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    "aria-hidden": true,
  };

  const moon = <path d="M20.5 14.8A8.2 8.2 0 0 1 9.2 3.5a8.2 8.2 0 1 0 11.3 11.3Z" />;

  switch (kind) {
    case "clear":
      return (
        <svg {...props} className={night ? "wx-icon wx-moon" : "wx-icon wx-sun"}>
          {night ? (
            moon
          ) : (
            <>
              <circle cx="12" cy="12" r="4" />
              <path d="M12 3v2M12 19v2M3 12h2M19 12h2M6 6l1.4 1.4M16.6 16.6 18 18M6 18l1.4-1.4M16.6 7.4 18 6" />
            </>
          )}
        </svg>
      );
    case "partly":
      return (
        <svg {...props} className={night ? "wx-icon wx-moon" : "wx-icon wx-sun"}>
          {night ? (
            <path d="M14.5 10.5a5.5 5.5 0 0 1-6.8-6.9 5.6 5.6 0 1 0 6.8 6.9Z" />
          ) : (
            <>
              <circle cx="8.5" cy="8.5" r="3" />
              <path d="M8.5 2.5v1.6M2.5 8.5h1.6M4.3 4.3l1.1 1.1M12.7 4.3l-1.1 1.1" />
            </>
          )}
          <path d="M9 20h8.5a3.5 3.5 0 0 0 .3-7 5 5 0 0 0-9.6 1.4A2.8 2.8 0 0 0 9 20Z" />
        </svg>
      );
    case "cloudy":
      return (
        <svg {...props} className="wx-icon wx-cloud">
          <path d="M7 18h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 18Z" />
        </svg>
      );
    case "fog":
      return (
        <svg {...props} className="wx-icon wx-fog">
          <path d="M7 13h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 13Z" />
          <path d="M5 17h14M8 20h9" />
        </svg>
      );
    case "snow":
      return (
        <svg {...props} className="wx-icon wx-snow">
          <path d="M7 14h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 14Z" />
          <path d="M8 18h.01M12 20h.01M16 18h.01" />
        </svg>
      );
    case "storm":
      return (
        <svg {...props} className="wx-icon wx-storm">
          <path d="M7 14h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 14Z" />
          <path d="M12.5 16 10 20h3l-1 3" />
        </svg>
      );
    default:
      return (
        <svg {...props} className="wx-icon wx-rain">
          <path d="M7 13h10a3.6 3.6 0 0 0 .4-7.2 5.4 5.4 0 0 0-10.3-.6A3.9 3.9 0 0 0 7 13Z" />
          <path d="M8.5 16.5 7.5 19M12 16.5 11 19M15.5 16.5 14.5 19" />
        </svg>
      );
  }
}
