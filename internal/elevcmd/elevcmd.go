package elevcmd

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
)

type ElevatorCommand struct {
	//Golang doesnt support union types,
	//so we have to pass any of the below
	//structs
	Value any
}

type DoorOpenCommand struct {
}

type DoorCloseCommand struct {
}

type StopLampCommand struct {
	Value bool
}

type MotorDirCommand struct {
	Dir elevconsts.Dirn
}

// Sets single button command
type ButtonLightCommand struct {
	Floor  int
	Button elevconsts.Button
	Value  bool
}

// Sets all button lights command
type ButtonLightArrayCommand struct {
	Array [elevconsts.N_FLOORS * elevconsts.N_BUTTONS]ButtonLightCommand
}

type FloorIndicatorCommand struct {
	Floor int
}

type RequestFloorCommand struct {
}

func (e *ElevatorCommand) CommandType() string {
	switch e.Value.(type) {
	case DoorOpenCommand:
		return "DoorOpenCommand"
	case DoorCloseCommand:
		return "DoorCloseCommand"
	case MotorDirCommand:
		return "MotorDirCommand"
	case ButtonLightCommand:
		return "ButtonLightCommand"
	case ButtonLightArrayCommand:
		return "ButtonLightArrayCommands"
	case FloorIndicatorCommand:
		return "FloorIndicatorCommand"
	case StopLampCommand:
		return "StopLampCommand"
	case RequestFloorCommand:
		return "RequestFloorCommand"
	default:
		return "UnknownCommand"
	}
}
