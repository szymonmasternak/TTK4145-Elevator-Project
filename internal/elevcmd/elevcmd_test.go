package elevcmd

import "testing"

func TestCommandType(t *testing.T) {
	elevatorCommandArray := []ElevatorCommand{
		{Value: DoorOpenCommand{}},
		{Value: DoorCloseCommand{}},
		{Value: MotorDirCommand{}},
		{Value: ButtonLightCommand{}},
		{Value: ButtonLightArrayCommand{}},
		{Value: FloorIndicatorCommand{}},
		{Value: StopLampCommand{}},
		{Value: RequestFloorCommand{}},
		{Value: struct{}{}},
	}

	elevatorCommandStringArray := []string{
		"DoorOpenCommand",
		"DoorCloseCommand",
		"MotorDirCommand",
		"ButtonLightCommand",
		"ButtonLightArrayCommands",
		"FloorIndicatorCommand",
		"StopLampCommand",
		"RequestFloorCommand",
		"UnknownCommand",
	}

	for index, elevatorCommand := range elevatorCommandArray {
		if elevatorCommand.CommandType() != elevatorCommandStringArray[index] {
			t.Errorf("Elevator.CommandType() returned %v, expected %v", elevatorCommand.CommandType(), elevatorCommandStringArray[index])
		}
	}
}
