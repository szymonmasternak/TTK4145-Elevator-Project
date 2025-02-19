package elevstate

import "github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"

// Struct equivalent to DirnBehaviourPair
type DirnBehaviourPair struct {
	Dirn      elevconsts.Dirn
	Behaviour elevconsts.ElevatorBehaviour
}

func (es *ElevatorState) requestsAbove() bool {
	for f := es.Floor + 1; f < elevconsts.N_FLOORS; f++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if es.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsBelow() bool {
	for f := 0; f < es.Floor; f++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if es.Requests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsHere() bool {
	for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
		if es.Requests[es.Floor][btn] != 0 {
			return true
		}
	}
	return false
}

// Determines the direction and behavior of the elevator
func (es *ElevatorState) RequestsChooseDirection() DirnBehaviourPair {
	switch es.Dirn {
	case elevconsts.Up:
		if es.requestsAbove() {
			return DirnBehaviourPair{elevconsts.Up, elevconsts.Moving}
		} else if es.requestsHere() {
			return DirnBehaviourPair{elevconsts.Down, elevconsts.DoorOpen}
		} else if es.requestsBelow() {
			return DirnBehaviourPair{elevconsts.Down, elevconsts.Moving}
		}
	case elevconsts.Down:
		if es.requestsBelow() {
			return DirnBehaviourPair{elevconsts.Down, elevconsts.Moving}
		} else if es.requestsHere() {
			return DirnBehaviourPair{elevconsts.Up, elevconsts.DoorOpen}
		} else if es.requestsAbove() {
			return DirnBehaviourPair{elevconsts.Up, elevconsts.Moving}
		}
	case elevconsts.Stop:
		if es.requestsHere() {
			return DirnBehaviourPair{elevconsts.Stop, elevconsts.DoorOpen}
		} else if es.requestsAbove() {
			return DirnBehaviourPair{elevconsts.Up, elevconsts.Moving}
		} else if es.requestsBelow() {
			return DirnBehaviourPair{elevconsts.Down, elevconsts.Moving}
		}
	}
	return DirnBehaviourPair{elevconsts.Stop, elevconsts.Idle}
}

func (es *ElevatorState) RequestsShouldStop() bool {
	switch es.Dirn {
	case elevconsts.Down:
		return es.Requests[es.Floor][elevconsts.HallDown] != 0 ||
			es.Requests[es.Floor][elevconsts.Cab] != 0 ||
			!es.requestsBelow()
	case elevconsts.Up:
		return es.Requests[es.Floor][elevconsts.HallUp] != 0 ||
			es.Requests[es.Floor][elevconsts.Cab] != 0 ||
			!es.requestsAbove()
	case elevconsts.Stop:
		return true
	}
	return true
}

func (es *ElevatorState) RequestsShouldClearImmediately(btnFloor int, btnType elevconsts.Button) bool {
	switch es.clearRequestVariant {
	case elevconsts.All:
		return es.Floor == btnFloor
	case elevconsts.InDirn:
		return es.Floor == btnFloor &&
			((es.Dirn == elevconsts.Up && btnType == elevconsts.HallUp) ||
				(es.Dirn == elevconsts.Down && btnType == elevconsts.HallDown) ||
				es.Dirn == elevconsts.Stop ||
				btnType == elevconsts.Cab)
	}
	return false
}

func (es *ElevatorState) RequestsClearAtCurrentFloor() {
	switch es.clearRequestVariant {
	case elevconsts.All:
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			es.Requests[es.Floor][btn] = 0
		}
	case elevconsts.InDirn:
		es.Requests[es.Floor][elevconsts.Cab] = 0
		switch es.Dirn {
		case elevconsts.Up:
			if !es.requestsAbove() && es.Requests[es.Floor][elevconsts.HallUp] == 0 {
				es.Requests[es.Floor][elevconsts.HallDown] = 0
			}
			es.Requests[es.Floor][elevconsts.HallUp] = 0
		case elevconsts.Down:
			if !es.requestsBelow() && es.Requests[es.Floor][elevconsts.HallDown] == 0 {
				es.Requests[es.Floor][elevconsts.HallUp] = 0
			}
			es.Requests[es.Floor][elevconsts.HallDown] = 0
		case elevconsts.Stop:
			es.Requests[es.Floor][elevconsts.HallUp] = 0
			es.Requests[es.Floor][elevconsts.HallDown] = 0
		}
	}
}
