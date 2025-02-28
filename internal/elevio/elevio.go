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
				Log.Debug().Msgf("Received channelButton")
				floor := buttonEvent.Floor
				button := elevconsts.Button(buttonEvent.Button)
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floor, Button: button}}
			case floor := <-elevio.channelFloor:
				Log.Debug().Msgf("Received channelFloor %v", floor)
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floor}}
			case <-elevio.channelStop:
				Log.Debug().Msgf("Received channelStop")
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.StopButtonEvent{}}
			case <-elevio.channelObstr:
				Log.Debug().Msgf("Received channelObstr")
				elevio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{}}
			default:
				if elevio.requestedFloor {
					Log.Debug().Msgf("Received requestedFloor")
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
					Log.Debug().Msgf("Received command %v", command.CommandType())
					elevio.driver.SetMotorDirection(MotorDirection(cmd.Dir))
				case elevcmd.ButtonLightCommand:
					elevio.driver.SetButtonLamp(ButtonType(cmd.Button), cmd.Floor, cmd.Value)
				case elevcmd.FloorIndicatorCommand:
					Log.Debug().Msgf("Received command %v", command.CommandType())
					elevio.driver.SetFloorIndicator(cmd.Floor)
				case elevcmd.DoorOpenCommand:
					Log.Debug().Msgf("Received command %v", command.CommandType())
					elevio.driver.SetDoorOpenLamp(true)
				case elevcmd.DoorCloseCommand:
					Log.Debug().Msgf("Received command %v", command.CommandType())
					elevio.driver.SetDoorOpenLamp(false)
				case elevcmd.StopLampCommand:
					Log.Debug().Msgf("Received command %v", command.CommandType())
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
