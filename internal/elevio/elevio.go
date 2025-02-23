package elevio

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type ElevIO interface {
	FloorIndicator(floor int)
	RequestButtonLight(floor int, button elevconsts.Button, value int)
	DoorLight(value int)
	StopButtonLight(value int)
	MotorDirection(direction elevconsts.Dirn)
}

type ElevatorIO struct {
	driver *ElevIODriver

	ChannelButton chan ButtonEvent
	ChannelFloor  chan int
	ChannelStop   chan bool
	ChannelObstr  chan bool
}

func NewElevatorIO(ipAddress string, numFloors int) (*ElevatorIO, error) {
	driver, err := NewElevIODriver(ipAddress, numFloors)

	if err != nil {
		Log.Error().Msgf("Error when creating elevator object %v", err)
		return nil, err
	}
	
	elevio := ElevatorIO{
		driver: driver,
		ChannelButton: make(chan ButtonEvent),
		ChannelFloor:  make(chan int),
		ChannelStop:   make(chan bool),
		ChannelObstr:  make(chan bool),
	}

	go driver.PollButtons(elevio.ChannelButton)
	go driver.PollFloorSensor(elevio.ChannelFloor)
	go driver.PollStopButton(elevio.ChannelStop)
	go driver.PollObstructionSwitch(elevio.ChannelObstr)
	
	return &elevio, nil
}

func (e *ElevatorIO) FloorIndicator(floor int) {
	e.driver.SetFloorIndicator(floor)
}

func (e *ElevatorIO) RequestButtonLight(floor int, button elevconsts.Button, value int) {
	e.driver.SetButtonLamp(ButtonType(button), floor, value != 0)
}

func (e *ElevatorIO) DoorLight(value int) {
	e.driver.SetDoorOpenLamp(value != 0)
}

func (e *ElevatorIO) StopButtonLight(value int) {
	e.driver.SetStopLamp(value != 0)
}

func (e *ElevatorIO) MotorDirection(direction elevconsts.Dirn) {
	e.driver.SetMotorDirection(MotorDirection(direction))
}

func (e *ElevatorIO) GetFloor() int {
	return e.driver.GetFloor()
}
