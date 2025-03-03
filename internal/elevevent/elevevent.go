package elevevent

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
)

type ElevatorEvent struct {
	//Golang doesnt support union types,
	//so we have to pass any of the below
	//structs
	Value any
}

type ButtonPressEvent struct {
	Floor  int
	Button elevconsts.Button //TODO: Delete ButtonType and replace with Button type?
}

func (bpe ButtonPressEvent) Wrap() ElevatorEvent {
	return ElevatorEvent{Value: bpe}
}

type FloorSensorEvent struct {
	Floor int
}

type StopButtonEvent struct {
	Value bool //Active-high
}

type ObstructionEvent struct {
	Value bool //Active-high
}

type RequestFloorEvent struct {
	Floor int
}

func (e *ElevatorEvent) EventType() string {
	switch e.Value.(type) {
	case ButtonPressEvent:
		return "ButtonEvent"
	case FloorSensorEvent:
		return "FloorSensorEvent"
	case StopButtonEvent:
		return "StopButtonEvent"
	case ObstructionEvent:
		return "ObstructionEvent"
	case RequestFloorEvent:
		return "RequestFloorEvent"
	default:
		return "UnknownEvent"
	}
}
