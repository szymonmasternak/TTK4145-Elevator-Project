package elevevent

import "testing"

func TestElevatorEvent(t *testing.T) {
	elevatorEventArray := []ElevatorEvent{
		{Value: ButtonPressEvent{}},
		{Value: FloorSensorEvent{}},
		{Value: StopButtonEvent{}},
		{Value: ObstructionEvent{}},
		{Value: RequestFloorEvent{}},
		{Value: struct{}{}},
	}

	elevatorEventStringArray := []string{
		"ButtonEvent",
		"FloorSensorEvent",
		"StopButtonEvent",
		"ObstructionEvent",
		"RequestFloorEvent",
		"UnknownEvent",
	}

	for index, elevatorEvent := range elevatorEventArray {
		if elevatorEvent.EventType() != elevatorEventStringArray[index] {
			t.Errorf("Elevator.EventType() returned %v, expected %v", elevatorEvent.EventType(), elevatorEventStringArray[index])
		}
	}
}
