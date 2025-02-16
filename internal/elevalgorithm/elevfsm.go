package elevalgorithm

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

// Elevator state variables
var elevator Elevator
var outputDevice ElevOutputDevice
var Log = logger.GetLogger() //valid across all files in node folder

func FSMinit() {
	elevator = elevator.Init(3.0, CV_InDirn) //Create uninitialized elevator
	elevator.Floor = GetFloor()              // Initialize elevator floor
	if elevator.Floor == -1 {
		Fsm_onInitBetweenFloors()
	}

	// Initialize output device
	outputDevice = GetOutputDevice()
}

// Set all button lights based on the elevator state
func setAllLights(es Elevator) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			outputDevice.RequestButtonLight(floor, Button(btn), es.Requests[floor][btn])
		}
	}
}

// Handles elevator initialization between floors
func Fsm_onInitBetweenFloors() {
	Log.Info().Msgf("Elevator initialized between floors, moving down to nearest floor")
	outputDevice.MotorDirection(D_Down)
	elevator.Dirn = D_Down
	elevator.Behaviour = EB_Moving
}

// Handles button press events
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
	logger.Log.Info().Msgf("\nFsm_onFloorArrival(%d)\n", newFloor)
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

	logger.Log.Info().Msgf("\nNew state:")
	elevator.Print()
}

// Handles door timeout event
func Fsm_onDoorTimeout() {
	logger.Log.Info().Msgf("\nFsm_onDoorTimeout() - Closing Door")
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

	logger.Log.Info().Msgf("\nNew state after door timeout:")
	elevator.Print()
}
