package elevator

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevio"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"

	"github.com/xyproto/randomstring"
)

var Logger = logger.GetLogger()

type Elevator struct {
	MetaData *elevmetadata.ElevMetaData //this contains all elevator constant metadata
	Network  *elevnet.ElevatorNetwork
	IO       *elevio.ElevatorIO
	State    *elevstate.ElevatorState

	initialised bool
}

func NewElevator(identifier string) *Elevator {
	if identifier == "" {
		identifier = randomstring.EnglishFrequencyString(10) //this should be random enough
		Logger.Warn().Msgf("No identifier provided, generated random identifier \"%v\"", identifier)
	}

	elevatorMetadata := &elevmetadata.ElevMetaData{
		SoftwareVersion: elevutils.GetGitHash(),
		IpAddress:       elevutils.GetLocalIP(),
		PortNumber:      9999,
		Identifier:      identifier,
	}

	elevIO := elevio.NewElevatorIO("localhost:15657", 4)
	elevState := elevstate.NewElevatorState(elevIO)

	return &Elevator{
		MetaData:    elevatorMetadata,
		Network:     elevnet.NewElevatorNetwork(elevatorMetadata),
		IO:          elevIO,
		State:       elevState,
		initialised: true,
	}
}

func (e *Elevator) Start() {
	// TODO setup all channels?
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
}

func (e *Elevator) Stop() {
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	// TODO
	// Terminate programme succesfully
	// Send message that programme is stopping
	// Close all sockets
	// etc
}
