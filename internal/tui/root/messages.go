package root

import tuiconfig "ekvs/internal/tui/config"

// triggerAuthMsg requests a transition to the auth screen.
// returnTo is the screen to return to after successful/cancelled auth.
type triggerAuthMsg struct{ returnTo screen }

// profileSwitchMsg requests a profile switch; the session is cleared.
type profileSwitchMsg struct{ profile tuiconfig.Profile }
