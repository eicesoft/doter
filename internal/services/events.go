package services

// EventOpenTab asks the frontend to pin the island and switch to a tab. The
// payload is a TabId string such as "overview", "agents" or "clipboard".
const EventOpenTab = "island:open-tab"

// EventAlert is emitted after the alert engine fires. The native notification
// is the primary delivery channel; the event keeps the UI in sync.
const EventAlert = "alerts:update"
