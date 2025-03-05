package elevio

import (
	"context"
	"sync"

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

	return &elevio, nil
}

func (eio *ElevatorIO) Start(ctx context.Context, waitGroup *sync.WaitGroup) {
	waitGroup.Add(2) //Two threads

	go eio.driver.PollButtons(eio.channelButton)
	go eio.driver.PollFloorSensor(eio.channelFloor)
	go eio.driver.PollStopButton(eio.channelStop)
	go eio.driver.PollObstructionSwitch(eio.channelObstr)

	go func() {
		defer waitGroup.Done()
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msgf("ElevatorIO driver channel Go routine has been signaled to stop")
				return
			case buttonEvent := <-eio.channelButton:
				Log.Debug().Msgf("Received channelButton")
				floor := buttonEvent.Floor
				button := elevconsts.Button(buttonEvent.Button)
				eio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floor, Button: button}}
			case floor := <-eio.channelFloor:
				Log.Debug().Msgf("Received channelFloor %v", floor)
				eio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floor}}
			case val := <-eio.channelStop:
				Log.Debug().Msgf("Received channelStop")
				eio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.StopButtonEvent{Value: val}}
			case val := <-eio.channelObstr:
				Log.Debug().Msgf("Received channelObstr")
				eio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{Value: val}}
			default:
				if eio.requestedFloor {
					Log.Debug().Msgf("Received requestedFloor")
					eio.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: eio.driver.GetFloor()}}
					eio.requestedFloor = false
				}
			}
		}
	}()

	go func() {
		defer waitGroup.Done()
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msgf("ElevatorIO Receive Command Channel Go routine has been signaled to stop")
				return
			case command := <-eio.commandChannel:
				Log.Debug().Msgf("Received command %v", command.CommandType())
				switch cmd := command.Value.(type) {
				case elevcmd.MotorDirCommand:
					eio.driver.SetMotorDirection(MotorDirection(cmd.Dir))
				case elevcmd.ButtonLightArrayCommand:
					for i := 0; i < len(cmd.Array); i++ {
						element := cmd.Array[i]
						eio.driver.SetButtonLamp(ButtonType(element.Button), element.Floor, element.Value)
					}
				case elevcmd.ButtonLightCommand:
					eio.driver.SetButtonLamp(ButtonType(cmd.Button), cmd.Floor, cmd.Value)
				case elevcmd.FloorIndicatorCommand:
					eio.driver.SetFloorIndicator(cmd.Floor)
				case elevcmd.DoorOpenCommand:
					eio.driver.SetDoorOpenLamp(true)
				case elevcmd.DoorCloseCommand:
					eio.driver.SetDoorOpenLamp(false)
				case elevcmd.StopLampCommand:
					eio.driver.SetStopLamp(cmd.Value)
				case elevcmd.RequestFloorCommand:
					eio.requestedFloor = true
				default:
					Log.Error().Msgf("Unknown command %v", cmd)
				}
			}
		}
	}()
}
