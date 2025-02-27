package elevstate

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
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

	//Internal Variables
	clearRequestVariant elevconsts.ClearRequestVariant
	doorOpenDuration    time.Duration
	doorCloseTime       time.Time
	eventChannel        <-chan elevevent.ElevatorEvent
	commandChannel      chan<- elevcmd.ElevatorCommand
}

func NewElevatorState(eventChannel <-chan elevevent.ElevatorEvent, commandChannel chan<- elevcmd.ElevatorCommand) *ElevatorState {
	elevatorState := &ElevatorState{
		Floor:               -1,
		Dirn:                elevconsts.Stop,
		Behaviour:           elevconsts.Idle,
		clearRequestVariant: elevconsts.InDirn, //TODO: Verify and maybe change?
		doorOpenDuration:    time.Second * 3,
		eventChannel:        eventChannel,
		commandChannel:      commandChannel,
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
					elevatorState.FsmOnFloorArrival(evnt.Floor)
				case elevevent.ButtonPressEvent:
					elevatorState.FsmOnRequestButtonPress(evnt.Floor, evnt.Button)
				case elevevent.StopButtonEvent:
					Log.Error().Msgf("StopButtonEvent not implemented")
				case elevevent.ObstructionEvent:
					Log.Error().Msgf("ObstructionEvent not implemented")
				case elevevent.RequestFloorEvent:
					Log.Error().Msgf("RequestFloorEvent should not occur")
				}

			}
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

func (es *ElevatorState) setAllLights() {
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.ButtonLightCommand{Floor: floor, Button: elevconsts.Button(btn), Value: es.Requests[floor][btn] != 0}}
		}
	}
}

func (es *ElevatorState) FsmOnRequestButtonPress(btn_floor int, btn_type elevconsts.Button) {
	Log.Info().Msgf("Button Has Been Pressed (%d, %s)", btn_floor, btn_type.String())

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		if es.RequestsShouldClearImmediately(btn_floor, btn_type) {
			es.doorCloseTime = time.Now().Add(es.doorOpenDuration)
		} else {
			es.Requests[btn_floor][btn_type] = 1
		}

	case elevconsts.Moving:
		es.Requests[btn_floor][btn_type] = 1

	case elevconsts.Idle:
		es.Requests[btn_floor][btn_type] = 1
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()

		switch es.Behaviour {
		case elevconsts.DoorOpen:

			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
			es.doorCloseTime = time.Now().Add(es.doorOpenDuration)
			es.RequestsClearAtCurrentFloor()

		case elevconsts.Moving:
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}

		case elevconsts.Idle:
			// Do nothing
		}
	}

	es.setAllLights()
}

// Handles elevator arrival at a new floor
func (es *ElevatorState) FsmOnFloorArrival(newFloor int) {
	es.Floor = newFloor

	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.FloorIndicatorCommand{Floor: es.Floor}}

	switch es.Behaviour {
	case elevconsts.Moving:
		if es.RequestsShouldStop() {

			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}}
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
			//elevator = es.RequestsClearAtCurrentFloor(elevator)
			es.RequestsClearAtCurrentFloor()
			// Timer_start(elevator.doorOpenDuration)
			es.doorCloseTime = time.Now().Add(es.doorOpenDuration)
			es.setAllLights()
			es.Behaviour = elevconsts.DoorOpen
		}
	}
}

// Handles door timeout event
func (es *ElevatorState) FsmOnDoorTimeout() {
	if es.Behaviour == elevconsts.DoorOpen {
		// Turn off door light (Close door)
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorCloseCommand{}}

		// Choose next action
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()

		switch es.Behaviour {
		case elevconsts.DoorOpen:
			// Restart timer if the elevator still has a request at this floor
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
			es.doorCloseTime = time.Now().Add(es.doorOpenDuration)
			// elevator = Requests_clearAtCurrentFloor(elevator)
			es.RequestsClearAtCurrentFloor()
			es.setAllLights()

		case elevconsts.Moving:
			// Move the elevator in the chosen direction
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}

		case elevconsts.Idle:
			// Do nothing, elevator stays idle
		}
	}
	Log.Info().Msg("New state after door timeout")
}
