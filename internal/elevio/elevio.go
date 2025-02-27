package elevio

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type ElevatorIO struct {
	//This Channel only Sends Events
	eventChannel   chan<- elevevent.ElevatorEvent
	commandChannel <-chan elevcmd.ElevatorCommand

	//Driver and driver channel instances
	driver        *ElevIODriver
	channelButton chan ButtonEvent
	channelFloor  chan int
	channelStop   chan bool
	channelObstr  chan bool

	//If a floor request has been sent
	requestedFloor bool
}

func NewElevatorIO(ipAddress string, numFloors int, eventChannel chan<- elevevent.ElevatorEvent, commandChannel <-chan elevcmd.ElevatorCommand) (*ElevatorIO, error) {
	driver, err := NewElevIODriver(ipAddress, numFloors)

	if err != nil {
		Log.Error().Msgf("Error when creating elevator object %v", err)
		return nil, err
	}

	elevio := ElevatorIO{
		driver:         driver,
		eventChannel:   eventChannel,
		commandChannel: commandChannel,
		channelButton:  make(chan ButtonEvent),
		channelFloor:   make(chan int),
		channelStop:    make(chan bool),
		channelObstr:   make(chan bool),

		requestedFloor: false,
	}

	go driver.PollButtons(elevio.channelButton)
	go driver.PollFloorSensor(elevio.channelFloor)
	go driver.PollStopButton(elevio.channelStop)
	go driver.PollObstructionSwitch(elevio.channelObstr)

	go func() {
		for {
			select {
			case buttonEvent := <-elevio.channelButton:
				floor := buttonEvent.Floor
				button := elevconsts.Button(buttonEvent.Button)
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floor, Button: button}}
			case floor := <-elevio.channelFloor:
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floor}}
			case <-elevio.channelStop:
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.StopButtonEvent{}}
			case <-elevio.channelObstr:
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{}}
			default:
				if elevio.requestedFloor {
					elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: elevio.driver.GetFloor()}}
					elevio.requestedFloor = false
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case command := <-elevio.commandChannel:
				switch cmd := command.Value.(type) {
				case elevcmd.MotorDirCommand:
					elevio.driver.SetMotorDirection(MotorDirection(cmd.Dir))
				case elevcmd.ButtonLightCommand:
					elevio.driver.SetButtonLamp(ButtonType(cmd.Button), cmd.Floor, cmd.Value)
				case elevcmd.FloorIndicatorCommand:
					elevio.driver.SetFloorIndicator(cmd.Floor)
				case elevcmd.DoorOpenCommand:
					elevio.driver.SetDoorOpenLamp(true)
				case elevcmd.DoorCloseCommand:
					elevio.driver.SetDoorOpenLamp(false)
				case elevcmd.StopLampCommand:
					elevio.driver.SetStopLamp(cmd.Value)
				case elevcmd.RequestFloorCommand:
					elevio.requestedFloor = true
				default:
					Log.Error().Msgf("Unknown command %v", cmd)
				}
			}
		}
	}()

	return &elevio, nil
}
