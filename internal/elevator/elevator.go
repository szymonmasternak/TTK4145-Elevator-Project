package elevator

import (
	"context"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevhallrequestassigner"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevio"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"

	"github.com/xyproto/randomstring"
)

var Logger = logger.GetLogger()

const (
	EVENT_CHANNEL_SIZE     = 10
	COMMAND_CHANNEL_SIZE   = 1
	IDENTIFIER_DEFAULT_LEN = 10
)

type Elevator struct {
	MetaData *elevmetadata.ElevMetaData //this contains all elevator constant metadata
	Network  *elevnet.ElevatorNetwork
	IO       *elevio.ElevatorIO
	State    *elevstate.ElevatorState
	//RequestStates       *requestconfirmation.RequestArray
	RequestHandler      *requestconfirmation.RequestHandler
	HallRequestAssigner *elevhallrequestassigner.HallRequestAssigner

	EventChannel   chan elevevent.ElevatorEvent
	CommandChannel chan elevcmd.ElevatorCommand

	StateOutChannel         chan elevstate.ElevatorState
	RequestUpdateChannel    chan requestconfirmation.RequestMessage
	localRequestsArrChannel chan requestconfirmation.RequestArray
	InboundReqArrayChannel  chan requestconfirmation.RequestArrayMessage
	OutboundReqArrayChannel chan requestconfirmation.RequestArrayMessage
	AlivePeersChannel       chan []string

	Initialised bool //set to true if initialised via NewElevator Function
	Running     bool

	//used for graceful shutdown
	waitGroupArray []*sync.WaitGroup
	cancelArray    []context.CancelFunc
}

func NewElevator(identifier string, portNumber uint16, driverIPAddress string, clearUpDownOnArrival bool) *Elevator {
	if identifier == "" {
		identifier = randomstring.EnglishFrequencyString(IDENTIFIER_DEFAULT_LEN) //this should be random enough
		Logger.Warn().Msgf("No elevator identifier provided, generated random identifier \"%v\"", identifier)
	}

	elevatorMetadata := &elevmetadata.ElevMetaData{
		SoftwareVersion: elevutils.GetGitHash(),
		IpAddress:       elevutils.GetLocalIP(),
		PortNumber:      portNumber,
		Identifier:      identifier,
	}

	eventChannel := make(chan elevevent.ElevatorEvent, EVENT_CHANNEL_SIZE)
	commandChannel := make(chan elevcmd.ElevatorCommand, COMMAND_CHANNEL_SIZE)
	stateInChannel := make(chan elevstate.ElevatorState, 10) //TODO: remove from everywhere
	stateOutChannel := make(chan elevstate.ElevatorState, 10)
	requestUpdatechannel := make(chan requestconfirmation.RequestMessage, 10)
	localRequestsArrChannel := make(chan requestconfirmation.RequestArray, 10)
	inboundReqArrayChannel := make(chan requestconfirmation.RequestArrayMessage, 10)
	outboundReqArrayChannel := make(chan requestconfirmation.RequestArrayMessage, 10)
	alivePeersChannel := make(chan []string, 10)

	elevIO, err := elevio.NewElevatorIO(driverIPAddress, elevconsts.N_FLOORS, eventChannel, commandChannel)
	if err != nil {
		panic("Error Creating ElevIO Object")
	}

	elevState := elevstate.NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival, stateInChannel, stateOutChannel, requestUpdatechannel)
	elevNetwork := elevnet.NewElevatorNetwork(elevatorMetadata, elevState, stateOutChannel, outboundReqArrayChannel, inboundReqArrayChannel, alivePeersChannel)
	elevAssigner := elevhallrequestassigner.NewHallRequestAssigner(elevatorMetadata.Identifier, elevNetwork.Listen, eventChannel, stateOutChannel, localRequestsArrChannel)
	reqHandler := requestconfirmation.NewRequestHandler(elevatorMetadata.Identifier, requestUpdatechannel, inboundReqArrayChannel, outboundReqArrayChannel, alivePeersChannel, localRequestsArrChannel)
	return &Elevator{
		MetaData:                elevatorMetadata,
		Network:                 elevNetwork,
		IO:                      elevIO,
		State:                   elevState,
		RequestHandler:          reqHandler,
		HallRequestAssigner:     elevAssigner,
		Initialised:             true,
		Running:                 false,
		EventChannel:            eventChannel,
		CommandChannel:          commandChannel,
		StateOutChannel:         stateOutChannel,
		RequestUpdateChannel:    requestUpdatechannel,
		InboundReqArrayChannel:  inboundReqArrayChannel,
		OutboundReqArrayChannel: outboundReqArrayChannel,
		AlivePeersChannel:       alivePeersChannel,
	}
}

func (e *Elevator) Start() {
	if !e.Initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	if e.Running {
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

	//Todo add other threads
	//Launch Threads One By One
	ctxAssigner, cancelAssigner := context.WithCancel(context.Background())
	wgAssigner := &sync.WaitGroup{}
	e.waitGroupArray = append(e.waitGroupArray, wgAssigner)
	e.HallRequestAssigner.Start(ctxAssigner, wgAssigner)
	e.cancelArray = append(e.cancelArray, cancelAssigner)
	e.RequestHandler.Start()
	e.Network.Broadcast.Start(time.Millisecond * 100)
	e.Network.Listen.Start()

	// go func() {
	// 	for {
	// 		select {
	// 		case requestArray := <-e.outboundReqArrayChannel:
	// 			if requestArray.Identifier == e.MetaData.Identifier {
	// 				for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
	// 					for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
	// 						if requestArray.RequestArray[floor][btn].State == requestconfirmation.REQ_Confirmed {
	// 							e.State.ConfirmedRequests[floor][btn] = 1
	// 						} else {
	// 							e.State.ConfirmedRequests[floor][btn] = 0
	// 						}
	// 					}
	// 				}
	// 				Logger.Debug().Msgf("Confirmed Requests: %v", e.State.ConfirmedRequests)
	// 			}
	// 		}
	// 	}
	// }()
}

func (e *Elevator) Stop() {
	if !e.Initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	if !e.Running {
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
	e.Running = false
}
