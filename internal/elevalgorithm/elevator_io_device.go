package elevalgorithm

// Wrapper functions for output handling
func wrap_requestButtonLight(floor int, button Button, value int) {
	SetButtonLamp(ButtonType(button), floor, value != 0)
}

func wrap_motorDirection(dirn Dirn) {
	SetMotorDirection(MotorDirection(dirn))
}

// Returns an instance of the output device
func GetOutputDevice() ElevOutputDevice {
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
