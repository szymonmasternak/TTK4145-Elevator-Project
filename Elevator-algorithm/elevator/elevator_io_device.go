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
		floor_sensor:   elevio.GetFloor,
		request_button: wrap_requestButton,
		stop_button:    func() int { return boolToInt(elevio.GetStop()) },
		obstruction:    func() int { return boolToInt(elevio.GetObstruction()) },
	}
}

// Returns an instance of the output device
func Elevio_getOutputDevice() ElevOutputDevice {
	return ElevOutputDevice{
		floor_indicator:      elevio.SetFloorIndicator,
		request_button_light: wrap_requestButtonLight,
		door_light:           func(value int) { elevio.SetDoorOpenLamp(value != 0) },
		stop_button_light:    func(value int) { elevio.SetStopLamp(value != 0) },
		motor_direction:      wrap_motorDirection,
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
