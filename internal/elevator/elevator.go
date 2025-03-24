package elevator

import (
	"context"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevio"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"

	"github.com/xyproto/randomstring"
)

var Logger = logger.GetLogger()

const (
	EVENT_CHANNEL_SIZE       = 10
	COMMAND_CHANNEL_SIZE     = 1
	IDENTIFIER_DEFAULT_LEN   = 10
	NETWORK_MSG_CHANNEL_SIZE = 1
)

type Elevator struct {
	MetaData *elevmetadata.ElevMetaData //this contains all elevator constant metadata
	Network  *elevnet.ElevatorNetwork
	IO       *elevio.ElevatorIO
	State    *elevstate.ElevatorState

	eventChannel   chan elevevent.ElevatorEvent
	commandChannel chan elevcmd.ElevatorCommand
	// networkMsgChannel chan elevnet.NetworkMsg

	initialised bool //set to true if initialised via NewElevator Function
	running     bool

	//used for graceful shutdown
	waitGroupArray []*sync.WaitGroup
	cancelArray    []context.CancelFunc
}

func NewElevator(identifier string, portNumber uint16, driverIPAddress string, clearUpDownOnArrival bool, udpPort uint16) *Elevator {
	if identifier == "" {
		identifier = randomstring.EnglishFrequencyString(IDENTIFIER_DEFAULT_LEN) //this should be random enough
		Logger.Warn().Msgf("No elevator identifier provided, generated random identifier \"%v\"", identifier)
	}

	elevatorMetadata := &elevmetadata.ElevMetaData{
		SoftwareVersion: elevutils.GetGitHash(),
		IpAddress:       elevutils.GetLocalIP(),
		PortNumber:      portNumber,
		Identifier:      identifier,
		UdpPort:         udpPort,
	}

	eventChannel := make(chan elevevent.ElevatorEvent, EVENT_CHANNEL_SIZE)
	commandChannel := make(chan elevcmd.ElevatorCommand, COMMAND_CHANNEL_SIZE)
	// networkMsgChannel := make(chan elevnet.NetworkMsg, NETWORK_MSG_CHANNEL_SIZE)

	elevIO, err := elevio.NewElevatorIO(driverIPAddress, elevconsts.N_FLOORS, eventChannel, commandChannel)
	if err != nil {
		panic("Error Creating ElevIO Object")
	}

	// elevState := elevstate.NewElevatorState(eventChannel, commandChannel, networkMsgChannel, clearUpDownOnArrival)
	// elevNetwork := elevnet.NewElevatorNetwork(elevatorMetadata, networkMsgChannel, elevState)

	elevState := elevstate.NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)
	elevNetwork := elevnet.NewElevatorNetwork(elevatorMetadata, elevState)

	return &Elevator{
		MetaData:       elevatorMetadata,
		Network:        elevNetwork,
		IO:             elevIO,
		State:          elevState,
		initialised:    true,
		running:        false,
		eventChannel:   eventChannel,
		commandChannel: commandChannel,
		// networkMsgChannel: networkMsgChannel,
	}
}

func (e *Elevator) Start() {
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	if e.running {
		Logger.Error().Msg("Elevator already running")
		return
	}

	//Launch Threads One By One
	ctxIO, cancelIO := context.WithCancel(context.Background())
	wgIO := &sync.WaitGroup{}
	e.waitGroupArray = append(e.waitGroupArray, wgIO)
	e.IO.Start(ctxIO, wgIO)
	e.cancelArray = append(e.cancelArray, cancelIO)

	//Launch Threads One by One
	ctxState, cancelState := context.WithCancel(context.Background())
	wgState := &sync.WaitGroup{}
	e.waitGroupArray = append(e.waitGroupArray, wgState)
	e.State.Start(ctxState, wgState)
	e.cancelArray = append(e.cancelArray, cancelState)

	ctxNetwork, cancelNetwork := context.WithCancel(context.Background())
	wgNetwork := &sync.WaitGroup{}
	e.waitGroupArray = append(e.waitGroupArray, wgNetwork)
	e.Network.Start(ctxNetwork, wgNetwork)
	e.cancelArray = append(e.cancelArray, cancelNetwork)

	//For Debug
	go func() {
		for {
			time.Sleep(time.Second)
			num := e.Network.GetNodesConnected()
			Logger.Info().Msgf("Elevators Connected: %d", num)
		}
	}()

	//Todo add other threads

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

	//Gracefully shutdown all threads one by one
	for i := len(e.cancelArray) - 1; i >= 0; i-- {
		e.cancelArray[i]()
		e.waitGroupArray[i].Wait()
	}

	Logger.Debug().Msg("Stopped Elevator")
	e.running = false
}
