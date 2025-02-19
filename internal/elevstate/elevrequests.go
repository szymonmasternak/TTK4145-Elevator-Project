package elevstate

// Struct equivalent to DirnBehaviourPair
type DirnBehaviourPair struct {
	Dirn      Dirn
	Behaviour ElevatorBehaviour
}

func (es *ElevatorState) requestsAbove() bool {
	for f := es.Floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if es.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsBelow() bool {
	for f := 0; f < es.Floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if es.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsHere() bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if es.Requests[es.Floor][btn] != 0 {
			return true
		}
	}
	return false
}

// Determines the direction and behavior of the elevator
func (es *ElevatorState) RequestsChooseDirection() DirnBehaviourPair {
	switch es.Dirn {
	case D_Up:
		if es.requestsAbove() {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if es.requestsHere() {
			return DirnBehaviourPair{D_Down, EB_DoorOpen}
		} else if es.requestsBelow() {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
	case D_Down:
		if es.requestsBelow() {
			return DirnBehaviourPair{D_Down, EB_Moving}
		} else if es.requestsHere() {
			return DirnBehaviourPair{D_Up, EB_DoorOpen}
		} else if es.requestsAbove() {
			return DirnBehaviourPair{D_Up, EB_Moving}
		}
	case D_Stop:
		if es.requestsHere() {
			return DirnBehaviourPair{D_Stop, EB_DoorOpen}
		} else if es.requestsAbove() {
			return DirnBehaviourPair{D_Up, EB_Moving}
		} else if es.requestsBelow() {
			return DirnBehaviourPair{D_Down, EB_Moving}
		}
	}
	return DirnBehaviourPair{D_Stop, EB_Idle}
}

func (es *ElevatorState) RequestsShouldStop() bool {
	switch es.Dirn {
	case D_Down:
		return es.Requests[es.Floor][B_HallDown] != 0 ||
			es.Requests[es.Floor][B_Cab] != 0 ||
			!es.requestsBelow()
	case D_Up:
		return es.Requests[es.Floor][B_HallUp] != 0 ||
			es.Requests[es.Floor][B_Cab] != 0 ||
			!es.requestsAbove()
	case D_Stop:
		return true
	}
	return true
}

func (es *ElevatorState) RequestsShouldClearImmediately(btnFloor int, btnType Button) bool {
	switch es.clearRequestVariant {
	case CV_All:
		return es.Floor == btnFloor
	case CV_InDirn:
		return es.Floor == btnFloor &&
			((es.Dirn == D_Up && btnType == B_HallUp) ||
				(es.Dirn == D_Down && btnType == B_HallDown) ||
				es.Dirn == D_Stop ||
				btnType == B_Cab)
	}
	return false
}

func (es *ElevatorState) RequestsClearAtCurrentFloor() {
	switch es.clearRequestVariant {
	case CV_All:
		for btn := 0; btn < N_BUTTONS; btn++ {
			es.Requests[es.Floor][btn] = 0
		}
	case CV_InDirn:
		es.Requests[es.Floor][B_Cab] = 0
		switch es.Dirn {
		case D_Up:
			if !es.requestsAbove() && es.Requests[es.Floor][B_HallUp] == 0 {
				es.Requests[es.Floor][B_HallDown] = 0
			}
			es.Requests[es.Floor][B_HallUp] = 0
		case D_Down:
			if !es.requestsBelow() && es.Requests[es.Floor][B_HallDown] == 0 {
				es.Requests[es.Floor][B_HallUp] = 0
			}
			es.Requests[es.Floor][B_HallDown] = 0
		case D_Stop:
			es.Requests[es.Floor][B_HallUp] = 0
			es.Requests[es.Floor][B_HallDown] = 0
		}
	}
}
