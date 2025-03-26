package elevstate

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
)

func (es *ElevatorState) requestsAbove() bool {
	for f := es.Floor + 1; f < elevconsts.N_FLOORS; f++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if es.ConfirmedRequests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsBelow() bool {
	for f := 0; f < es.Floor; f++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if es.ConfirmedRequests[f][btn] != 0 {
				return true
			}
		}
	}
	return false
}

func (es *ElevatorState) requestsHere() bool {
	for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
		if es.ConfirmedRequests[es.Floor][btn] != 0 {
			return true
		}
	}
	return false
}

// Determines the direction and behavior of the elevator
func (es *ElevatorState) RequestsChooseDirection() (elevconsts.Dirn, elevconsts.ElevatorBehaviour) {
	switch es.Dirn {
	case elevconsts.UP:
		switch {
		case es.requestsAbove():
			return elevconsts.UP, elevconsts.MOVING
		case es.requestsHere():
			return elevconsts.DOWN, elevconsts.DOOR_OPEN
		case es.requestsBelow():
			return elevconsts.DOWN, elevconsts.MOVING
		}
	case elevconsts.DOWN:
		switch {
		case es.requestsBelow():
			return elevconsts.DOWN, elevconsts.MOVING
		case es.requestsHere():
			return elevconsts.UP, elevconsts.DOOR_OPEN
		case es.requestsAbove():
			return elevconsts.UP, elevconsts.MOVING
		}
	case elevconsts.STOP:
		switch {
		case es.requestsHere():
			return elevconsts.STOP, elevconsts.DOOR_OPEN
		case es.requestsAbove():
			return elevconsts.UP, elevconsts.MOVING
		case es.requestsBelow():
			return elevconsts.DOWN, elevconsts.MOVING
		}
	}
	return elevconsts.STOP, elevconsts.IDLE
}

func (es *ElevatorState) RequestsShouldStop() bool {
	switch es.Dirn {
	case elevconsts.DOWN:
		return es.ConfirmedRequests[es.Floor][elevconsts.HALL_DOWN] != 0 ||
			es.ConfirmedRequests[es.Floor][elevconsts.CAB] != 0 ||
			!es.requestsBelow()
	case elevconsts.UP:
		return es.ConfirmedRequests[es.Floor][elevconsts.HALL_UP] != 0 ||
			es.ConfirmedRequests[es.Floor][elevconsts.CAB] != 0 ||
			!es.requestsAbove()
	case elevconsts.STOP:
		return true
	}
	return true
}

func (es *ElevatorState) RequestsShouldClearImmediately(btnFloor int, btnType elevconsts.Button) bool {
	switch es.clearRequestVariant {
	case elevconsts.ALL:
		return es.Floor == btnFloor
	case elevconsts.IN_DIRN:
		return es.Floor == btnFloor &&
			((es.Dirn == elevconsts.UP && btnType == elevconsts.HALL_UP) ||
				(es.Dirn == elevconsts.DOWN && btnType == elevconsts.HALL_DOWN) ||
				es.Dirn == elevconsts.STOP ||
				btnType == elevconsts.CAB)
	}
	return false
}

func (es *ElevatorState) RequestsClearAtCurrentFloor() {
	switch es.clearRequestVariant {
	case elevconsts.ALL:
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.Button(btn),
				State:  requestconfirmation.REQ_Completed,
			}

		}
	case elevconsts.IN_DIRN:
		es.ConfirmedRequests[es.Floor][elevconsts.CAB] = 0
		switch es.Dirn {
		case elevconsts.UP:
			if !es.requestsAbove() && es.ConfirmedRequests[es.Floor][elevconsts.HALL_UP] == 0 {
				es.updateRequestChannel <- requestconfirmation.RequestMessage{
					Floor:  es.Floor,
					Button: elevconsts.HALL_DOWN,
					State:  requestconfirmation.REQ_Completed,
				}
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HALL_UP,
				State:  requestconfirmation.REQ_Completed,
			}
		case elevconsts.DOWN:
			if !es.requestsBelow() && es.ConfirmedRequests[es.Floor][elevconsts.HALL_DOWN] == 0 {
				es.updateRequestChannel <- requestconfirmation.RequestMessage{
					Floor:  es.Floor,
					Button: elevconsts.HALL_UP,
					State:  requestconfirmation.REQ_Completed,
				}
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HALL_DOWN,
				State:  requestconfirmation.REQ_Completed,
			}
		case elevconsts.STOP:
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HALL_UP,
				State:  requestconfirmation.REQ_Completed,
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HALL_DOWN,
				State:  requestconfirmation.REQ_Completed,
			}
		}
	}
}
