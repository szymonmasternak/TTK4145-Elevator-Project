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
	running		bool
	stopChannel chan bool //sending true to this channel will stop the elevator
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

	elevIO, err := elevio.NewElevatorIO("localhost:15657", 4)
	if err != nil {
		panic("Error Creating ElevIO Object")
	}

	elevState := elevstate.NewElevatorState(elevIO)

	return &Elevator{
		MetaData:    elevatorMetadata,
		Network:     elevnet.NewElevatorNetwork(elevatorMetadata),
		IO:          elevIO,
		State:       elevState,
		initialised: true,
		running:     false,
	}
}

func (e *Elevator) Start() {
	// TODO setup all channels?
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	if e.running {
		Logger.Error().Msg("Elevator already running")
		return
	}

	go func() {
		for{
			select {
			case <-e.stopChannel:
				e.running = false;
				return
			}
		}
	}()

	e.running = true
}

func (e *Elevator) Stop() {
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	if !e.running {
		Logger.Error().Msg("Elevator not running, so cannot stop elevator")
		return
	}

	Logger.Debug().Msg("Stopping Elevator")

	e.stopChannel <- true
}
