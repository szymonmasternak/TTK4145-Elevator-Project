package elevstate

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevio"
)

var Log = logger.GetLogger()

type ElevatorState struct {
	Floor     int
	Dirn      Dirn
	Requests  [N_FLOORS][N_BUTTONS]int
	Behaviour ElevatorBehaviour

	//Internal Variables
	clearRequestVariant ClearRequestVariant
	doorOpenDuration_s  time.Duration

	io *elevio.ElevatorIO //internal pointer
}

func NewElevatorState(ioDriver *elevio.ElevatorIO) *ElevatorState {
	elevatorState := &ElevatorState{
		Floor:               -1,
		Dirn:                D_Stop,
		Behaviour:           EB_Idle,
		clearRequestVariant: CV_InDirn, //TODO: Verify and maybe change?
		doorOpenDuration_s:  time.Second * 3,
		io:                  ioDriver,
	}
	elevatorState.Floor = elevatorState.io.GetFloor()

	if elevatorState.Floor == -1 {
		Log.Info().Msgf("Elevator initialized between floors, moving down to nearest floor")
		elevatorState.io.MotorDirection(D_Down)
		elevatorState.Dirn = D_Down
		elevatorState.Behaviour = EB_Moving
	}

	return elevatorState
}

func (es *ElevatorState) Print() {
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |floor = %-2d          |\n"+
		"  |dirn  = %-12s|\n"+
		"  |behav = %-12s|\n",
		es.Floor,
		es.Dirn.ToString(),
		ElevatorBehaviour.toString(es.Behaviour),
	)
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |  | up  | dn  | cab |")

	for f := N_FLOORS - 1; f >= 0; f-- {
		Log.Info().Msgf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && btn == int(B_HallUp)) || (f == 0 && btn == int(B_HallDown)) {
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

func setAllLights(es Elevator) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			outputDevice.RequestButtonLight(floor, Button(btn), es.Requests[floor][btn])
		}
	}
}

func Fsm_onRequestButtonPress(btn_floor int, btn_type Button) {
	Log.Info().Msgf("\nFsm_onRequestButtonPress(%d, %s)\n", btn_floor, btn_type.ToString())
	elevator.Print()

	switch elevator.Behaviour {
	case EB_DoorOpen:
		if Requests_shouldClearImmediately(elevator, btn_floor, btn_type) {
			Timer_start(elevator.doorOpenDuration_s)
		} else {
			elevator.Requests[btn_floor][btn_type] = 1
		}

	case EB_Moving:
		elevator.Requests[btn_floor][btn_type] = 1

	case EB_Idle:
		elevator.Requests[btn_floor][btn_type] = 1
		pair := Requests_chooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case EB_DoorOpen:
			outputDevice.DoorLight(1)
			Timer_start(elevator.doorOpenDuration_s)
			elevator = Requests_clearAtCurrentFloor(elevator)

		case EB_Moving:
			outputDevice.MotorDirection(elevator.Dirn)

		case EB_Idle:
			// Do nothing
		}
	}

	setAllLights(elevator)

	Log.Info().Msgf("\nNew state:")
	elevator.Print()
}

// Handles elevator arrival at a new floor
func Fsm_onFloorArrival(newFloor int) {
	Log.Info().Msgf("\nFsm_onFloorArrival(%d)\n", newFloor)
	elevator.Print()

	elevator.Floor = newFloor
	outputDevice.FloorIndicator(elevator.Floor)

	switch elevator.Behaviour {
	case EB_Moving:
		if Requests_shouldStop(elevator) {
			outputDevice.MotorDirection(D_Stop)
			outputDevice.DoorLight(1)
			elevator = Requests_clearAtCurrentFloor(elevator)
			Timer_start(elevator.doorOpenDuration_s)
			setAllLights(elevator)
			elevator.Behaviour = EB_DoorOpen
		}
	}

	Log.Info().Msgf("\nNew state:")
	elevator.Print()
}

// Handles door timeout event
func Fsm_onDoorTimeout() {
	Log.Info().Msgf("\nFsm_onDoorTimeout() - Closing Door")
	elevator.Print()

	if elevator.Behaviour == EB_DoorOpen {
		// Turn off door light (Close door)
		outputDevice.DoorLight(0)

		// Choose next action
		pair := Requests_chooseDirection(elevator)
		elevator.Dirn = pair.Dirn
		elevator.Behaviour = pair.Behaviour

		switch elevator.Behaviour {
		case EB_DoorOpen:
			// Restart timer if the elevator still has a request at this floor
			outputDevice.DoorLight(1)
			Timer_start(elevator.doorOpenDuration_s)
			elevator = Requests_clearAtCurrentFloor(elevator)
			setAllLights(elevator)

		case EB_Moving:
			// Move the elevator in the chosen direction
			outputDevice.MotorDirection(elevator.Dirn)

		case EB_Idle:
			// Do nothing, elevator stays idle
		}
	}

	Log.Info().Msgf("\nNew state after door timeout:")
	elevator.Print()
}
