package elevstatenetmsg

import "github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"

// Putting this in elevconsts file looks off, it also looks off here...
// Needs to be in a seperate file due to import cycle
type ElevatorStateNetMsg struct {
	//variables from state->network
	Floor          int
	Button         elevconsts.Button
	TimeoutOccured bool //false by default

	//variables from network->state
	ShouldDoRequest bool
}
