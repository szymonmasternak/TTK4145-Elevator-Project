package elevalgorithm


// Wrapper functions for input handling
func wrap_requestButton(floor int, button Button) int {
	if GetButton(ButtonType(button), floor) {
		return 1
	}
	return 0
}

// Wrapper functions for output handling
func wrap_requestButtonLight(floor int, button Button, value int) {
	SetButtonLamp(ButtonType(button), floor, value != 0)
}

func wrap_motorDirection(dirn Dirn) {
	SetMotorDirection(MotorDirection(dirn))
}

// Returns an instance of the input device
func Elevio_getInputDevice() ElevInputDevice {
	return ElevInputDevice{
		FloorSensor:   GetFloor,
		RequestButton: wrap_requestButton,
		StopButton:    func() int { return boolToInt(GetStop()) },
		Obstruction:   func() int { return boolToInt(GetObstruction()) },
	}
}

// Returns an instance of the output device
func Elevio_getOutputDevice() ElevOutputDevice {
	return ElevOutputDevice{
		FloorIndicator:     SetFloorIndicator,
		RequestButtonLight: wrap_requestButtonLight,
		DoorLight:          func(value int) { SetDoorOpenLamp(value != 0) },
		StopButtonLight:    func(value int) { SetStopLamp(value != 0) },
		MotorDirection:     wrap_motorDirection,
	}
}

// Converts Dirn (direction) to a string
func (d Dirn) ToString() string {
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
func (b Button) ToString() string {
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
