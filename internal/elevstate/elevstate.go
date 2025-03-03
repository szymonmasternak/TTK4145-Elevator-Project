package elevstate

import (
	"encoding/json"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
)

var Log = logger.GetLogger()

type ElevatorState struct {
	Floor     int
	Dirn      elevconsts.Dirn
	Requests  [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int
	Behaviour elevconsts.ElevatorBehaviour

	MetaData *elevmetadata.ElevMetaData // ✅ NEW: Reference metadata (which includes the identifier)

	// Internal Variables
	clearRequestVariant elevconsts.ClearRequestVariant
	obstructionSensor   bool
	stopButton          bool
	doorOpenDuration    time.Duration
	doorOpenTime        time.Time
	eventChannel        <-chan elevevent.ElevatorEvent
	commandChannel      chan<- elevcmd.ElevatorCommand

	// Outbound: ElevatorState updates to be broadcast.
	stateOutChannel chan<- ElevatorState
	// Inbound: Remote ElevatorState updates coming from the network.
	stateInChannel <-chan ElevatorState
}

func serializeState(state ElevatorState) ([]byte, error) {
	return json.Marshal(state)
}

func deserializeState(data []byte) (ElevatorState, error) {
	var state ElevatorState
	err := json.Unmarshal(data, &state)
	return state, err
}

func NewElevatorState(
	metaData *elevmetadata.ElevMetaData, // ✅ Pass metadata instead of a separate identifier
	eventChannel <-chan elevevent.ElevatorEvent,
	commandChannel chan<- elevcmd.ElevatorCommand,
	stateOutChannel chan<- ElevatorState,
	stateInChannel <-chan ElevatorState,
) *ElevatorState {
	elevatorState := &ElevatorState{
		Floor:               -1,
		Dirn:                elevconsts.Stop,
		Behaviour:           elevconsts.Idle,
		MetaData:            metaData, // ✅ Assign metadata
		clearRequestVariant: elevconsts.InDirn,
		doorOpenDuration:    time.Second * 3,
		eventChannel:        eventChannel,
		commandChannel:      commandChannel,
		stopButton:          false,
		obstructionSensor:   false,
		doorOpenTime:        time.Time{},
		stateOutChannel:     stateOutChannel,
		stateInChannel:      stateInChannel,
	}

	elevatorState.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.RequestFloorCommand{}}

	//TODO: Add a timeout event to this for safety
	for {
		event := <-eventChannel
		req, ok := event.Value.(elevevent.RequestFloorEvent)
		if ok {
			elevatorState.Floor = req.Floor
			break
		}
	}

	if elevatorState.Floor == -1 {
		Log.Info().Msgf("Elevator initialized between floors, moving down to nearest floor")
		elevatorState.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Down}}
		elevatorState.Dirn = elevconsts.Down
		elevatorState.Behaviour = elevconsts.Moving
	}

	go func() {
		for {
			select {
			case event := <-elevatorState.eventChannel:
				switch evnt := event.Value.(type) {
				case elevevent.FloorSensorEvent:
					elevatorState.handleFloorArrival(evnt.Floor)
				case elevevent.ButtonPressEvent:
					Log.Info().Msgf("Button Pressed: (%d, %s)", evnt.Button, evnt.Button.String())
					elevatorState.handleButtonPress(evnt.Floor, evnt.Button)
				case elevevent.StopButtonEvent:
					Log.Info().Msgf("Stop Button: %v", evnt.Value)
					elevatorState.handleStopButton(evnt.Value)
				case elevevent.ObstructionEvent:
					Log.Info().Msgf("Obstruction: %v", evnt.Value)
					elevatorState.handleObstruction(evnt.Value)
				case elevevent.RequestFloorEvent:
					Log.Error().Msgf("RequestFloorEvent should not occur")
				}

				// ✅ **Send the updated state to stateOutChannel after processing an event**
				select {
				case elevatorState.stateOutChannel <- *elevatorState:
					Log.Debug().Msg("Sent Elevator State update to stateOutChannel")
				default:
					// Drop update if channel is full (non-blocking send)
				}

			default:
				if time.Now().After(elevatorState.doorOpenTime.Add(elevatorState.doorOpenDuration)) {
					if elevatorState.Behaviour == elevconsts.DoorOpen && !elevatorState.stopButton {
						Log.Warn().Msgf("Door timeout Event")
						elevatorState.handleDoorTimeout()

						// ✅ **Send the updated state to stateOutChannel after a door timeout**
						select {
						case elevatorState.stateOutChannel <- *elevatorState:
							Log.Debug().Msg("Sent Elevator State update due to door timeout")
						default:
						}
					}
				}
			}
		}
	}()

	go func() {
		for remoteState := range stateInChannel {
			// ✅ Ignore state updates from itself using MetaData.Identifier
			if remoteState.MetaData.Identifier == elevatorState.MetaData.Identifier {
				continue
			}

			Log.Debug().Msgf("Received remote state update: %+v", remoteState)
			// TODO: Merge or process the remote state
		}
	}()

	return elevatorState
}

func (es *ElevatorState) Print() {
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |floor = %-2d          |\n"+
		"  |dirn  = %-12s|\n"+
		"  |behav = %-12s|\n",
		es.Floor,
		es.Dirn.String(),
		es.Behaviour.String(),
	)
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |  | up  | dn  | cab |")

	for f := elevconsts.N_FLOORS - 1; f >= 0; f-- {
		Log.Info().Msgf("  | %d", f)
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if (f == elevconsts.N_FLOORS-1 && btn == int(elevconsts.HallUp)) || (f == 0 && btn == int(elevconsts.HallDown)) {
				Log.Info().Msgf("|     ")
			} else {
				if es.Requests[f][btn] != 0 {
					Log.Info().Msgf("|  #  ")
				} else {
					Log.Info().Msgf("|  -  ")
				}
			}
		}
		Log.Info().Msgf("|")
	}
	Log.Info().Msgf("  +--------------------+")
}

func (es *ElevatorState) setAllLightsSequence() {
	var buttonArray [elevconsts.N_FLOORS * elevconsts.N_BUTTONS]elevcmd.ButtonLightCommand

	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			i := floor*elevconsts.N_BUTTONS + btn
			buttonArray[i] = elevcmd.ButtonLightCommand{
				Floor:  floor,
				Button: elevconsts.Button(btn),
				Value:  es.Requests[floor][btn] != 0,
			}
		}
	}

	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.ButtonLightArrayCommand{Array: buttonArray}}
}

func (es *ElevatorState) handleButtonPress(btnFloor int, btnType elevconsts.Button) {
	if es.stopButton {
		Log.Warn().Msgf("Stop Button Pressed, not responding to button presses")
		return
	}

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		if es.RequestsShouldClearImmediately(btnFloor, btnType) {
			es.doorOpenTime = time.Now().Add(es.doorOpenDuration)
		} else {
			es.Requests[btnFloor][btnType] = 1
		}

	case elevconsts.Moving:
		es.Requests[btnFloor][btnType] = 1

	case elevconsts.Idle:
		es.Requests[btnFloor][btnType] = 1
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()

		switch es.Behaviour {
		case elevconsts.DoorOpen:
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
			es.doorOpenTime = time.Now()
			es.RequestsClearAtCurrentFloor()

		case elevconsts.Moving:
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
		}
	}

	es.setAllLightsSequence()
}

// Handles elevator arrival at a new floor
func (es *ElevatorState) handleFloorArrival(newFloor int) {
	es.Floor = newFloor
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.FloorIndicatorCommand{Floor: es.Floor}}

	if es.Behaviour == elevconsts.Moving && es.RequestsShouldStop() {
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}}
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}

		es.doorOpenTime = time.Now()
		es.RequestsClearAtCurrentFloor()
		es.setAllLightsSequence()
		es.Behaviour = elevconsts.DoorOpen
	}
}

// Handles door timeout event
func (es *ElevatorState) handleDoorTimeout() {
	if es.obstructionSensor && es.Behaviour == elevconsts.DoorOpen {
		Log.Warn().Msgf("Obstruction Detected, not trying to close door for another %v", es.doorOpenDuration.String())
		es.doorOpenTime = time.Now()
		return
	}

	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorCloseCommand{}}
	es.Dirn, es.Behaviour = es.RequestsChooseDirection()

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
		es.doorOpenTime = time.Now()
		es.RequestsClearAtCurrentFloor()
		es.setAllLightsSequence()
	case elevconsts.Moving:
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
	}
}

func (es *ElevatorState) handleStopButton(stopButtonState bool) {
	es.stopButton = stopButtonState

	if es.stopButton {
		es.Dirn = elevconsts.Stop
	} else {
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()
	}
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
}

func (es *ElevatorState) handleObstruction(obstructionState bool) {
	es.obstructionSensor = obstructionState
}
