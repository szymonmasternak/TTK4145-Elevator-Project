package elevstate

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevio"
)

var Log = logger.GetLogger()

type ElevatorState struct {
	Floor     int
	Dirn      elevconsts.Dirn
	Requests  [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int
	Behaviour elevconsts.ElevatorBehaviour

	//Internal Variables
	clearRequestVariant elevconsts.ClearRequestVariant
	doorOpenDuration_s  time.Duration

	io *elevio.ElevatorIO //internal pointer

	endTime time.Time //We need to figure out if this is really needed
}

func NewElevatorState(ioDriver *elevio.ElevatorIO) *ElevatorState {
	elevatorState := &ElevatorState{
		Floor:               -1,
		Dirn:                elevconsts.Stop,
		Behaviour:           elevconsts.Idle,
		clearRequestVariant: elevconsts.InDirn, //TODO: Verify and maybe change?
		doorOpenDuration_s:  time.Second * 3,
		io:                  ioDriver,
	}
	elevatorState.Floor = elevatorState.io.GetFloor()

	if elevatorState.Floor == -1 {
		Log.Info().Msgf("Elevator initialized between floors, moving down to nearest floor")
		elevatorState.io.MotorDirection(elevconsts.Down)
		elevatorState.Dirn = elevconsts.Down
		elevatorState.Behaviour = elevconsts.Moving
	}

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
			es.io.RequestButtonLight(floor, elevconsts.Button(btn), es.Requests[floor][btn])
		}
	}
}

func (es *ElevatorState) FsmOnRequestButtonPress(btn_floor int, btn_type elevconsts.Button) {
	Log.Info().Msgf("Fsm_onRequestButtonPress(%d, %s)\n", btn_floor, btn_type.String())

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		if es.RequestsShouldClearImmediately(btn_floor, btn_type) {
			es.endTime = time.Now().Add(es.doorOpenDuration_s)
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
			es.io.DoorLight(1)
			es.endTime = time.Now().Add(es.doorOpenDuration_s)
			es.RequestsClearAtCurrentFloor()

		case elevconsts.Moving:
			es.io.MotorDirection(es.Dirn)

		case elevconsts.Idle:
			// Do nothing
		}
	}

	es.setAllLights()
	Log.Info().Msgf("New state:")
}

// Handles elevator arrival at a new floor
func (es *ElevatorState) FsmOnFloorArrival(newFloor int) {
	Log.Info().Msgf("Fsm_onFloorArrival(%d)\n", newFloor)

	es.Floor = newFloor
	es.io.FloorIndicator(es.Floor)

	switch es.Behaviour {
	case elevconsts.Moving:
		if es.RequestsShouldStop() {
			es.io.MotorDirection(elevconsts.Stop)
			es.io.DoorLight(1)
			//elevator = es.RequestsClearAtCurrentFloor(elevator)
			es.RequestsClearAtCurrentFloor()
			// Timer_start(elevator.doorOpenDuration_s)
			es.endTime = time.Now().Add(es.doorOpenDuration_s)
			es.setAllLights()
			es.Behaviour = elevconsts.DoorOpen
		}
	}
}

// Handles door timeout event
func (es *ElevatorState) FsmOnDoorTimeout() {
	Log.Info().Msg("Closing Door")

	if es.Behaviour == elevconsts.DoorOpen {
		// Turn off door light (Close door)
		es.io.DoorLight(0)

		// Choose next action
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()

		switch es.Behaviour {
		case elevconsts.DoorOpen:
			// Restart timer if the elevator still has a request at this floor
			es.io.DoorLight(1)
			es.endTime = time.Now().Add(es.doorOpenDuration_s)
			// elevator = Requests_clearAtCurrentFloor(elevator)
			es.RequestsClearAtCurrentFloor()
			es.setAllLights()

		case elevconsts.Moving:
			// Move the elevator in the chosen direction
			es.io.MotorDirection(es.Dirn)

		case elevconsts.Idle:
			// Do nothing, elevator stays idle
		}
	}
	Log.Info().Msg("New state after door timeout")
}
