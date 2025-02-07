package elevalgorithm

// Struct equivalent to DirnBehaviourPair
type DirnBehaviourPair struct {
	Dirn      Dirn
	Behaviour ElevatorBehaviour
}


// Checks if there are requests above the current floor
func requests_above(e Elevator) bool {
	for f := e.Floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

// Checks if there are requests below the current floor
func requests_below(e Elevator) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

// Checks if there are requests at the current floor
func requests_here(e Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.Requests[e.Floor][btn] != 0 {
			return true
		}
	}
	return false
}

// Determines the direction and behavior of the elevator
func Requests_chooseDirection(e Elevator) DirnBehaviourPair {
	switch e.Dirn {
	case D_Up:
		if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if requests_here(e) {
			return DirnBehaviourPair{D_Down, EB_DoorOpen}
		} else if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
	case D_Down:
		if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		} else if requests_here(e) {
			return DirnBehaviourPair{D_Up, EB_DoorOpen}
		} else if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
	case D_Stop:
		if requests_here(e) {
			return DirnBehaviourPair{D_Stop, EB_DoorOpen}
		} else if requests_above(e) {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if requests_below(e) {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
	}
	return DirnBehaviourPair{D_Stop, EB_Idle}
}

// Determines if the elevator should stop at the current floor
func Requests_shouldStop(e Elevator) bool {
	switch e.Dirn {
	case D_Down:
		return e.Requests[e.Floor][B_HallDown] != 0 ||
			e.Requests[e.Floor][B_Cab] != 0 ||
			!requests_below(e)
	case D_Up:
		return e.Requests[e.Floor][B_HallUp] != 0 ||
			e.Requests[e.Floor][B_Cab] != 0 ||
			!requests_above(e)
	case D_Stop:
		return true
	}
	return true
}

// Determines if a request should be cleared immediately
func Requests_shouldClearImmediately(e Elevator, btnFloor int, btnType Button) bool {
	switch e.clearRequestVariant {
	case CV_All:
		return e.Floor == btnFloor
	case CV_InDirn:
		return e.Floor == btnFloor &&
			((e.Dirn == D_Up && btnType == B_HallUp) ||
				(e.Dirn == D_Down && btnType == B_HallDown) ||
				e.Dirn == D_Stop ||
				btnType == B_Cab)
	}
	return false
}

// Clears requests at the current floor
func Requests_clearAtCurrentFloor(e Elevator) Elevator {
	switch e.clearRequestVariant {
	case CV_All:
		for btn := 0; btn < N_BUTTONS; btn++ {
			e.Requests[e.Floor][btn] = 0
		}
	case CV_InDirn:
		e.Requests[e.Floor][B_Cab] = 0
		switch e.Dirn {
		case D_Up:
			if !requests_above(e) && e.Requests[e.Floor][B_HallUp] == 0 {
				e.Requests[e.Floor][B_HallDown] = 0
			}
			e.Requests[e.Floor][B_HallUp] = 0
		case D_Down:
			if !requests_below(e) && e.Requests[e.Floor][B_HallDown] == 0 {
				e.Requests[e.Floor][B_HallUp] = 0
			}
			e.Requests[e.Floor][B_HallDown] = 0
		case D_Stop:
			e.Requests[e.Floor][B_HallUp] = 0
			e.Requests[e.Floor][B_HallDown] = 0
		}
	}
	return e
}
