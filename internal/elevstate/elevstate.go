package elevstate

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstatenetmsg"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
)

var Log = logger.GetLogger()

type ElevatorState struct {
	Floor     int
	Dirn      elevconsts.Dirn
	Requests  [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int
	Behaviour elevconsts.ElevatorBehaviour

	//Internal Variables
	clearRequestVariant elevconsts.ClearRequestVariant
	obstructionSensor   bool
	stopButton          bool
	doorOpenDuration    time.Duration
	doorOpenTime        time.Time
	eventChannel        <-chan elevevent.ElevatorEvent
	commandChannel      chan<- elevcmd.ElevatorCommand
	stateNetChannel     chan elevstatenetmsg.ElevatorStateNetMsg
}

func NewElevatorState(eventChannel <-chan elevevent.ElevatorEvent, commandChannel chan<- elevcmd.ElevatorCommand, clearUpDownOnArrival bool, stateNetChannel chan elevstatenetmsg.ElevatorStateNetMsg) *ElevatorState {
	clearRequestVariant := elevconsts.InDirn
	if clearUpDownOnArrival {
		clearRequestVariant = elevconsts.All
	}

	elevatorState := &ElevatorState{
		Floor:               -1,
		Dirn:                elevconsts.Stop,
		Behaviour:           elevconsts.Idle,
		clearRequestVariant: clearRequestVariant,
		doorOpenDuration:    time.Second * 3,
		eventChannel:        eventChannel,
		commandChannel:      commandChannel,
		stopButton:          false,
		obstructionSensor:   false,
		doorOpenTime:        time.Time{}, //Returns zero value, since we dont know when it was last open

		stateNetChannel: stateNetChannel,
	}
	return elevatorState
}

func (es *ElevatorState) Start(ctx context.Context, waitGroup *sync.WaitGroup) error {
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.RequestFloorCommand{}}

	timeout := time.After(500 * time.Millisecond) //This number looks nice
	select {
	case <-ctx.Done():
		Log.Warn().Msgf("ElevatorState Start has been signaled to stop")
		return nil
	case <-timeout:
		return errors.New("ElevatorState Start timed out")
	case event := <-es.eventChannel:
		req, ok := event.Value.(elevevent.RequestFloorEvent)
		if ok {
			es.Floor = req.Floor
			break
		}
	}

	if es.Floor == -1 {
		Log.Info().Msgf("Elevator initialized between floors, moving down to nearest floor")
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Down}}
		es.Dirn = elevconsts.Down
		es.Behaviour = elevconsts.Moving
	}

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msgf("ElevatorState Go routine has been signaled to stop")
				if es.Dirn != elevconsts.Stop {
					Log.Warn().Msgf("Elevator is not stopped, stopping it")
					es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}}
				}
				return
			case event := <-es.eventChannel:
				switch evnt := event.Value.(type) {
				case elevevent.FloorSensorEvent:
					es.handleFloorArrival(evnt.Floor)
				case elevevent.ButtonPressEvent:
					Log.Info().Msgf("Button Has Been Pressed (%d, %s)", evnt.Button, evnt.Button.String())
					es.handleButtonPress(evnt.Floor, evnt.Button, false)
				case elevevent.StopButtonEvent:
					Log.Info().Msgf("Stop Button is %v", evnt.Value)
					es.handleStopButton(evnt.Value)
				case elevevent.ObstructionEvent:
					Log.Info().Msgf("Obstruction Button is %v", evnt.Value)
					es.handleObstruction(evnt.Value)
				case elevevent.RequestFloorEvent:
					Log.Error().Msgf("RequestFloorEvent should not occur")
				case elevevent.NetworkButtonEvent:
					Log.Info().Msgf("Network Button Pressed (%d, %s)", evnt.Floor, evnt.Button.String())
					es.handleButtonPress(evnt.Floor, evnt.Button, true)
				}
			default:
				if time.Now().After(es.doorOpenTime.Add(es.doorOpenDuration)) {
					if es.Behaviour == elevconsts.DoorOpen {
						if !es.stopButton {
							Log.Warn().Msgf("Door timeout Event")
							es.handleDoorTimeout()
						}
					}
				}
			}
		}
	}()
	return nil
}

func (es *ElevatorState) Print() {
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |floor = %-2d          |\n"+
		"  |dirn  = %-12s|\n"+
		"  |behav = %-12s|\n",
		es.Floor,
		es.Dirn.String(),
		es.Behaviour.String(),
	)
	Log.Info().Msgf("  +--------------------+")
	Log.Info().Msgf("  |  | up  | dn  | cab |")

	for f := elevconsts.N_FLOORS - 1; f >= 0; f-- {
		Log.Info().Msgf("  | %d", f)
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if (f == elevconsts.N_FLOORS-1 && btn == int(elevconsts.HallUp)) || (f == 0 && btn == int(elevconsts.HallDown)) {
				Log.Info().Msgf("|     ")
			} else {
				if es.Requests[f][btn] != 0 {
					Log.Info().Msgf("|  #  ")
				} else {
					Log.Info().Msgf("|  -  ")
				}
			}
		}
		Log.Info().Msgf("|")
	}
	Log.Info().Msgf("  +--------------------+")
}

func (es *ElevatorState) setAllLightsSequence() {
	var buttonArray [elevconsts.N_FLOORS * elevconsts.N_BUTTONS]elevcmd.ButtonLightCommand

	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			i := floor*elevconsts.N_BUTTONS + btn
			buttonArray[i] = elevcmd.ButtonLightCommand{
				Floor:  floor,
				Button: elevconsts.Button(btn),
				Value:  es.Requests[floor][btn] != 0,
			}
		}
	}

	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.ButtonLightArrayCommand{Array: buttonArray}}
}

func (es *ElevatorState) handleButtonPress(btnFloor int, btnType elevconsts.Button, fromNetwork bool) {
	if es.stopButton {
		Log.Warn().Msgf("Stop Button Pressed, not responding to button presses")
		return
	}

	if !fromNetwork {
		if btnType == elevconsts.HallDown || btnType == elevconsts.HallUp {
			es.stateNetChannel <- elevstatenetmsg.ElevatorStateNetMsg{Floor: btnFloor, Button: btnType, TimeoutOccured: false}

			select {
			case msg := <-es.stateNetChannel:
				if !msg.ShouldDoRequest {
					Log.Warn().Msgf("Network Module decided that the button should be served by another node")
					return
				} else {
					Log.Warn().Msgf("Network Module decided that the button should be served locally")
				}
			case <-time.After(time.Second): //TODO fix
				Log.Warn().Msgf("Timeout, network module timeout, continuing to serve locally")
				es.stateNetChannel <- elevstatenetmsg.ElevatorStateNetMsg{Floor: btnFloor, Button: btnType, TimeoutOccured: true}
			}
		}
	}

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		if es.RequestsShouldClearImmediately(btnFloor, btnType) {
			es.doorOpenTime = time.Now().Add(es.doorOpenDuration)
		} else {
			es.Requests[btnFloor][btnType] = 1
		}

	case elevconsts.Moving:
		es.Requests[btnFloor][btnType] = 1

	case elevconsts.Idle:
		es.Requests[btnFloor][btnType] = 1
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()

		switch es.Behaviour {
		case elevconsts.DoorOpen:
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
			es.doorOpenTime = time.Now()
			es.RequestsClearAtCurrentFloor()

		case elevconsts.Moving:
			es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
		}
	}

	es.setAllLightsSequence()
}

// Handles elevator arrival at a new floor
func (es *ElevatorState) handleFloorArrival(newFloor int) {
	es.Floor = newFloor
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.FloorIndicatorCommand{Floor: es.Floor}}

	if es.Behaviour == elevconsts.Moving && es.RequestsShouldStop() {
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}}
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}

		es.doorOpenTime = time.Now()
		es.RequestsClearAtCurrentFloor()
		es.setAllLightsSequence()
		es.Behaviour = elevconsts.DoorOpen
	}
}

// Handles door timeout event
func (es *ElevatorState) handleDoorTimeout() {
	if es.obstructionSensor && es.Behaviour == elevconsts.DoorOpen {
		Log.Warn().Msgf("Obstruction Detected, not trying to close door for another %v", es.doorOpenDuration.String())
		es.doorOpenTime = time.Now()
		return
	}

	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorCloseCommand{}}
	es.Dirn, es.Behaviour = es.RequestsChooseDirection()

	switch es.Behaviour {
	case elevconsts.DoorOpen:
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.DoorOpenCommand{}}
		es.doorOpenTime = time.Now()
		es.RequestsClearAtCurrentFloor()
		es.setAllLightsSequence()
	case elevconsts.Moving:
		es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
	}
}

func (es *ElevatorState) handleStopButton(stopButtonState bool) {
	es.stopButton = stopButtonState
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.StopLampCommand{Value: es.stopButton}}

	if es.stopButton {
		es.Dirn = elevconsts.Stop
	} else {
		es.Dirn, es.Behaviour = es.RequestsChooseDirection()
	}
	es.commandChannel <- elevcmd.ElevatorCommand{Value: elevcmd.MotorDirCommand{Dir: es.Dirn}}
}

func (es *ElevatorState) handleObstruction(obstructionState bool) {
	es.obstructionSensor = obstructionState
}

func (es ElevatorState) CalculateTimeToServeReq(Floor int, Button elevconsts.Button) time.Duration {
	if Floor < 0 || Button > elevconsts.N_BUTTONS {
		Log.Error().Msgf("Floor or Button out of range")
		return time.Hour * 5
	}
	if es.Floor < 0 {
		Log.Error().Msgf("es.Floor out of range")
		return time.Hour * 5
	}

	esCopy := es
	esCopy.Requests[Floor][Button] = 1

	arrivedAtRequest := false
	checkTargetRequest := func(clearedFloor int, clearedBtn elevconsts.Button) {
		if clearedFloor == Floor && clearedBtn == Button {
			arrivedAtRequest = true
		}
	}
	var duration time.Duration
	if es.Behaviour == elevconsts.DoorOpen {
		duration += time.Hour //Penalty to not assign orders to open doors :-)
	}
	if es.stopButton {
		duration += time.Hour * 100 //Big penalty never assign requests to stopped elevators
	}

	switch esCopy.Behaviour {
	case elevconsts.Idle:
		esCopy.Dirn, _ = esCopy.RequestsChooseDirection()
		if esCopy.Dirn == elevconsts.Stop {
			return duration
		}
	case elevconsts.Moving:
		duration += elevconsts.ELEVATOR_TRAVEL_DURATION
		esCopy.Floor += int(esCopy.Dirn)
	case elevconsts.DoorOpen:
		duration -= es.doorOpenDuration / 2
	}

	for {
		if esCopy.RequestsShouldStop() {
			esCopy.RequestsClearAtCurrentFloorErrorCheck(checkTargetRequest)

			if arrivedAtRequest {
				Log.Debug().Msgf("Elevator at floor %d Duration calculated: %s", es.Floor, duration)
				return duration
			}
			duration += es.doorOpenDuration
			esCopy.Dirn, _ = esCopy.RequestsChooseDirection()
		}
		esCopy.Floor += int(esCopy.Dirn)
		duration += elevconsts.ELEVATOR_TRAVEL_DURATION
	}
	return duration
}
