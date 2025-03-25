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
	case elevconsts.Up:
		switch {
		case es.requestsAbove():
			return elevconsts.Up, elevconsts.Moving
		case es.requestsHere():
			return elevconsts.Down, elevconsts.DoorOpen
		case es.requestsBelow():
			return elevconsts.Down, elevconsts.Moving
		}
	case elevconsts.Down:
		switch {
		case es.requestsBelow():
			return elevconsts.Down, elevconsts.Moving
		case es.requestsHere():
			return elevconsts.Up, elevconsts.DoorOpen
		case es.requestsAbove():
			return elevconsts.Up, elevconsts.Moving
		}
	case elevconsts.Stop:
		switch {
		case es.requestsHere():
			return elevconsts.Stop, elevconsts.DoorOpen
		case es.requestsAbove():
			return elevconsts.Up, elevconsts.Moving
		case es.requestsBelow():
			return elevconsts.Down, elevconsts.Moving
		}
	}
	return elevconsts.Stop, elevconsts.Idle
}

func (es *ElevatorState) RequestsShouldStop() bool {
	switch es.Dirn {
	case elevconsts.Down:
		return es.ConfirmedRequests[es.Floor][elevconsts.HallDown] != 0 ||
			es.ConfirmedRequests[es.Floor][elevconsts.Cab] != 0 ||
			!es.requestsBelow()
	case elevconsts.Up:
		return es.ConfirmedRequests[es.Floor][elevconsts.HallUp] != 0 ||
			es.ConfirmedRequests[es.Floor][elevconsts.Cab] != 0 ||
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
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.Button(btn),
				State:  requestconfirmation.REQ_Completed,
			}

		}
	case elevconsts.InDirn:
		es.ConfirmedRequests[es.Floor][elevconsts.Cab] = 0
		switch es.Dirn {
		case elevconsts.Up:
			if !es.requestsAbove() && es.ConfirmedRequests[es.Floor][elevconsts.HallUp] == 0 {
				es.updateRequestChannel <- requestconfirmation.RequestMessage{
					Floor:  es.Floor,
					Button: elevconsts.HallDown,
					State:  requestconfirmation.REQ_Completed,
				}
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HallUp,
				State:  requestconfirmation.REQ_Completed,
			}
		case elevconsts.Down:
			if !es.requestsBelow() && es.ConfirmedRequests[es.Floor][elevconsts.HallDown] == 0 {
				es.updateRequestChannel <- requestconfirmation.RequestMessage{
					Floor:  es.Floor,
					Button: elevconsts.HallUp,
					State:  requestconfirmation.REQ_Completed,
				}
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HallDown,
				State:  requestconfirmation.REQ_Completed,
			}
		case elevconsts.Stop:
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HallUp,
				State:  requestconfirmation.REQ_Completed,
			}
			es.updateRequestChannel <- requestconfirmation.RequestMessage{
				Floor:  es.Floor,
				Button: elevconsts.HallDown,
				State:  requestconfirmation.REQ_Completed,
			}
		}
	}
}
