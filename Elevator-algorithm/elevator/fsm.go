package elevator

import (
	"fmt"
)

// Elevator state variables
var elevator Elevator
var outputDevice ElevOutputDevice

// Equivalent of `__attribute__((constructor)) fsm_init()`
func init() {
	elevator = elevator_uninitialized()

	elevator.Config.doorOpenDuration_s = 3.0
	elevator.Config.clearRequestVariant = CV_InDirn

	// Initialize output device
	outputDevice = Elevio_getOutputDevice()
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
	outputDevice.MotorDirection(D_Down)
	elevator.Dirn = D_Down
	elevator.Behaviour = EB_Moving
}

// Handles button press events
func Fsm_onRequestButtonPress(btn_floor int, btn_type Button) {
	fmt.Printf("\n\nFsm_onRequestButtonPress(%d, %s)\n", btn_floor, Elevio_button_toString(btn_type))
	elevator_print(elevator)

	switch elevator.Behaviour {
	case EB_DoorOpen:
		if Requests_shouldClearImmediately(elevator, btn_floor, btn_type) {
			Timer_start(elevator.Config.doorOpenDuration_s)
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
			Timer_start(elevator.Config.doorOpenDuration_s)
			elevator = Requests_clearAtCurrentFloor(elevator)

		case EB_Moving:
			outputDevice.MotorDirection(elevator.Dirn)

		case EB_Idle:
			// Do nothing
		}
	}

	setAllLights(elevator)

	fmt.Println("\nNew state:")
	elevator_print(elevator)
}

// Handles elevator arrival at a new floor
func Fsm_onFloorArrival(newFloor int) {
	fmt.Printf("\n\nFsm_onFloorArrival(%d)\n", newFloor)
	elevator_print(elevator)

	elevator.Floor = newFloor
	outputDevice.FloorIndicator(elevator.Floor)

	switch elevator.Behaviour {
	case EB_Moving:
		if Requests_shouldStop(elevator) {
			outputDevice.MotorDirection(D_Stop)
			outputDevice.DoorLight(1)
			elevator = Requests_clearAtCurrentFloor(elevator)
			Timer_start(elevator.Config.doorOpenDuration_s)
			setAllLights(elevator)
			elevator.Behaviour = EB_DoorOpen
		}
	}

	fmt.Println("\nNew state:")
	elevator_print(elevator)
}

// Handles door timeout event
func Fsm_onDoorTimeout() {
	fmt.Println("\n\nFsm_onDoorTimeout() - Closing Door")
	elevator_print(elevator)

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
			Timer_start(elevator.Config.doorOpenDuration_s)
			elevator = Requests_clearAtCurrentFloor(elevator)
			setAllLights(elevator)

		case EB_Moving:
			// Move the elevator in the chosen direction
			outputDevice.MotorDirection(elevator.Dirn)

		case EB_Idle:
			// Do nothing, elevator stays idle
		}
	}

	fmt.Println("\nNew state after door timeout:")
	elevator_print(elevator)
}
