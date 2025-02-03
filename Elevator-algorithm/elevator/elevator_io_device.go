package elevator

import "elevator-algorithm/elevio"

// Wrapper functions for input handling
func wrap_requestButton(floor int, button Button) int {
	if elevio.GetButton(elevio.ButtonType(button), floor) {
		return 1
	}
	return 0
}

// Wrapper functions for output handling
func wrap_requestButtonLight(floor int, button Button, value int) {
	elevio.SetButtonLamp(elevio.ButtonType(button), floor, value != 0)
}

func wrap_motorDirection(dirn Dirn) {
	elevio.SetMotorDirection(elevio.MotorDirection(dirn))
}

// Returns an instance of the input device
func Elevio_getInputDevice() ElevInputDevice {
	return ElevInputDevice{
		FloorSensor:   elevio.GetFloor,
		RequestButton: wrap_requestButton,
		StopButton:    func() int { return boolToInt(elevio.GetStop()) },
		Obstruction:   func() int { return boolToInt(elevio.GetObstruction()) },
	}
}

// Returns an instance of the output device
func Elevio_getOutputDevice() ElevOutputDevice {
	return ElevOutputDevice{
		FloorIndicator:     elevio.SetFloorIndicator,
		RequestButtonLight: wrap_requestButtonLight,
		DoorLight:          func(value int) { elevio.SetDoorOpenLamp(value != 0) },
		StopButtonLight:    func(value int) { elevio.SetStopLamp(value != 0) },
		MotorDirection:     wrap_motorDirection,
	}
}

// Converts Dirn (direction) to a string
func Elevio_dirn_toString(d Dirn) string {
	switch d {
	case D_Up:
		return "D_Up"
	case D_Down:
		return "D_Down"
	case D_Stop:
		return "D_Stop"
	default:
		return "D_UNDEFINED"
	}
}

// Converts Button type to a string
func Elevio_button_toString(b Button) string {
	switch b {
	case B_HallUp:
		return "B_HallUp"
	case B_HallDown:
		return "B_HallDown"
	case B_Cab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

// Helper function to convert bool to int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
