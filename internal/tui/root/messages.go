package root

import tuiconfig "ekvs/internal/tui/config"

// triggerAuthMsg requests a transition to the auth screen.
// returnTo is the screen to return to after successful/cancelled auth.
type triggerAuthMsg struct{ returnTo screen }

// triggerProjectsMsg requests navigation to the Projects screen.
// If the session is not authenticated, auth is triggered first.
type triggerProjectsMsg struct{}

// triggerProfilesMsg requests navigation to the Profiles screen from the main menu.
type triggerProfilesMsg struct{}

// profileSwitchMsg requests a profile switch; the session is cleared.
type profileSwitchMsg struct{ profile tuiconfig.Profile }
