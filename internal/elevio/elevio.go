package elevio

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type ElevIO interface {
	FloorIndicator(floor int)
	RequestButtonLight(floor int, button elevstate.Button, value int)
	DoorLight(value int)
	StopButtonLight(value int)
	MotorDirection(direction elevstate.Dirn)
}

type ElevatorIO struct {
	driver *ElevIODriver
}

func NewElevatorIO(ipAddress string, numFloors int) *ElevatorIO {
	driver, err := NewElevIODriver(ipAddress, numFloors)

	if err != nil {
		Log.Error().Msgf("Error when creating elevator object %v", err)
	}

	return &ElevatorIO{
		driver: driver,
	}
}

func (e *ElevatorIO) FloorIndicator(floor int) {
	e.driver.SetFloorIndicator(floor)
}

func (e *ElevatorIO) RequestButtonLight(floor int, button elevstate.Button, value int) {
	e.driver.SetButtonLamp(ButtonType(button), floor, value != 0)
}

func (e *ElevatorIO) DoorLight(value int) {
	e.driver.SetDoorOpenLamp(value != 0)
}

func (e *ElevatorIO) StopButtonLight(value int) {
	e.driver.SetStopLamp(value != 0)
}

func (e *ElevatorIO) MotorDirection(direction elevstate.Dirn) {
	e.driver.SetMotorDirection(MotorDirection(direction))
}

func (e *ElevatorIO) GetFloor() int {
	return e.driver.GetFloor()
}
