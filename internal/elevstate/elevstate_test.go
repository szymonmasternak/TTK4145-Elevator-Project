package elevstate

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevcmd"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

const TEST_DELAY = 100 * time.Millisecond

func waitForCommands(channel <-chan elevcmd.ElevatorCommand, duration time.Duration) []elevcmd.ElevatorCommand {
	var cmds []elevcmd.ElevatorCommand
	timeout := time.After(duration)
	for {
		select {
		case cmd := <-channel:
			cmds = append(cmds, cmd)
		case <-timeout:
			return cmds
		}
	}
}

func serialiseCommand(cmd elevcmd.ElevatorCommand) string {
	b, err := json.Marshal(cmd)
	if err != nil {
		return ""
	}
	return string(b)
}

func indexOfCommand(cmds []elevcmd.ElevatorCommand, target elevcmd.ElevatorCommand) int {
	for i, cmd := range cmds {
		if serialiseCommand(cmd) == serialiseCommand(target) {
			return i
		}
	}
	return -1
}

// checks order of commands
func checkOrder(t *testing.T, actual, expected []elevcmd.ElevatorCommand) {
	lastIndex := -1
	for _, cmd := range expected {
		idx := indexOfCommand(actual, cmd)
		if idx == -1 {
			t.Errorf("Expected command %v but not found in actual sequence: %v", cmd, actual)
			return
		}
		if idx < lastIndex {
			t.Errorf("Command %v appears out of order: %v", cmd, actual)
			return
		}
		lastIndex = idx
	}
}

func TestElevatorStateInitialisation(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 10)
	commandChannel := make(chan elevcmd.ElevatorCommand, 1)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 3
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	go func() {
		for {
			<-commandChannel
		}
	}()

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
	if elevState.Floor != floorStart {
		t.Errorf("Expected floor to be %v, got %d", floorStart, elevState.Floor)
	}
}

// Tests that events propagate to state object
func TestNewElevatorStateCommands(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 10)
	commandChannel := make(chan elevcmd.ElevatorCommand, 1)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 3
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	go func() {
		for {
			<-commandChannel
		}
	}()

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected error to be nil, got %v", err)
	}
	if elevState.Floor != floorStart {
		t.Errorf("Expected floor to be %v, got %v", floorStart, elevState.Floor)
	}

	// Test All Buttons
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floor, Button: elevconsts.Button(btn)}}
			time.Sleep(TEST_DELAY)
			if elevState.Requests[floor][btn] != 1 {
				t.Errorf("Expected request for floor %v and button %v to be 1, got %v", floor, btn, elevState.Requests[floor][btn])
			}
		}
	}

	sensorEventFloor := 2
	// Test floor sensor event.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: sensorEventFloor}}
	time.Sleep(TEST_DELAY)
	if elevState.Floor != sensorEventFloor {
		t.Errorf("Expected floor to be %v, got %v", sensorEventFloor, elevState.Floor)
	}

	// Test Stop Button event.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.StopButtonEvent{Value: true}}
	time.Sleep(TEST_DELAY)
	if !elevState.stopButton {
		t.Errorf("Expected stop button to be true, got %v", elevState.stopButton)
	}
	if elevState.Dirn != elevconsts.Stop {
		t.Errorf("Expected direction to be Stop, got %v", elevState.Dirn)
	}

	// Test obstruction event.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{Value: true}}
	time.Sleep(TEST_DELAY)
	if !elevState.obstructionSensor {
		t.Errorf("Expected obstruction sensor to be true, got %v", elevState.obstructionSensor)
	}
}

// Checks timeout functionality if floor is not provided
func TestElevatorStateStartTimeout(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 10)
	commandChannel := make(chan elevcmd.ElevatorCommand, 1)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	err := elevState.Start(ctx, wg)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// Test Door Open Duration
func TestDoorOpenDuration(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 1
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	stopClearingChannel <- true

	if elevState.Floor != floorStart {
		t.Errorf("Expected floor to be %v, got %v", floorStart, elevState.Floor)
	}

	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorStart, Button: elevconsts.HallUp}}
	time.Sleep(TEST_DELAY)
	if elevState.Behaviour != elevconsts.DoorOpen {
		t.Errorf("Expected door to be open immediately after button press")
	}

	// Check first command is DoorOpen
	initialCmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, initialCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.DoorOpenCommand{}},
	})

	// Check doorCloses after certain time
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	timeoutCmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, timeoutCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.DoorCloseCommand{}},
	})
}

// Test obstruction on elevator
func TestObstruction(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 1
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	stopClearingChannel <- true

	time.Sleep(TEST_DELAY)
	if elevState.Behaviour != elevconsts.Idle {
		t.Errorf("Expected elevator to be idle by default %v", elevState.Behaviour)
	}

	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorStart, Button: elevconsts.HallUp}}
	time.Sleep(TEST_DELAY)
	if elevState.Behaviour != elevconsts.DoorOpen {
		t.Errorf("Expected door to be open immediately after button press")
	}

	// Check first command is DoorOpen
	initialCmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, initialCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.DoorOpenCommand{}},
	})

	// Send obstruction event so door should remain open.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{Value: true}}
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	cmdsDuringObstruction := waitForCommands(commandChannel, TEST_DELAY)
	doorCloseCommand := elevcmd.ElevatorCommand{Value: elevcmd.DoorCloseCommand{}}
	// Check that DoorCloseCommand does NOT appear.
	for _, cmd := range cmdsDuringObstruction {
		if cmd == doorCloseCommand {
			t.Error("Found DoorCloseCommand while obstructed")
		}
	}

	if elevState.obstructionSensor != true {
		t.Errorf("Expected obstruction to be true")
	}

	// Clear the obstruction and expect a DoorCloseCommand.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ObstructionEvent{Value: false}}
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	cmdsAfterClear := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, cmdsAfterClear, []elevcmd.ElevatorCommand{doorCloseCommand})
}

// Test Stop Button Functionality
func TestStopButton(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 0
	floorButtonRequest := 3
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	stopClearingChannel <- true

	time.Sleep(TEST_DELAY)
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorButtonRequest, Button: elevconsts.Cab}}
	cmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, cmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Up}},
	})

	// Simulate stop button press.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.StopButtonEvent{Value: true}}
	time.Sleep(TEST_DELAY)

	cmds2 := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, cmds2, []elevcmd.ElevatorCommand{
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}},
	})
	if !elevState.stopButton {
		t.Errorf("Expected stop button to be true")
	}
	if elevState.Dirn != elevconsts.Stop {
		t.Errorf("Expected elevator direction to be Stop, got %d", elevState.Dirn)
	}
}

// TestCabCall. Sends elevator from 0 to 3 floor and checks commands
func TestCabCall(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 0
	floorButtonRequest := 3
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	stopClearingChannel <- true

	// Simulate cab call for floor 3.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorButtonRequest, Button: elevconsts.Cab}}
	time.Sleep(TEST_DELAY)
	if elevState.Requests[floorButtonRequest][elevconsts.Cab] != 1 {
		t.Errorf("Expected cab call for floor %v to be registered", floorButtonRequest)
	}

	for len(commandChannel) > 0 {
		<-commandChannel
	}

	// Simulate arrival at floor 3.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floorButtonRequest}}
	time.Sleep(TEST_DELAY)
	arrivalCmds := waitForCommands(commandChannel, TEST_DELAY)
	expectedSequence := []elevcmd.ElevatorCommand{
		{Value: elevcmd.FloorIndicatorCommand{Floor: floorButtonRequest}},
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}},
		{Value: elevcmd.DoorOpenCommand{}},
	}
	checkOrder(t, arrivalCmds, expectedSequence)

	// Wait for door to close
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	if elevState.Requests[floorButtonRequest][elevconsts.Cab] != 0 {
		t.Errorf("Expected cab call for floor %v to be cleared after arrival; got %v", floorButtonRequest, elevState.Requests[floorButtonRequest][elevconsts.Cab])
	}
}

// Test Hall Call of Elevator
func TestHallCall(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 1
	floorButtonRequest := 2
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	stopClearingChannel <- true

	// Simulate hall call
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorButtonRequest, Button: elevconsts.HallUp}}
	time.Sleep(TEST_DELAY)
	if elevState.Requests[floorButtonRequest][elevconsts.HallUp] != 1 {
		t.Errorf("Expected hall call for floor %v (HallUp) to be registered in state", floorButtonRequest)
	}

	// Simulate arrival at floor
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floorButtonRequest}}
	time.Sleep(TEST_DELAY)
	arrivalCmds := waitForCommands(commandChannel, TEST_DELAY)
	expectedSequence := []elevcmd.ElevatorCommand{
		{Value: elevcmd.FloorIndicatorCommand{Floor: floorButtonRequest}},
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}},
	}
	checkOrder(t, arrivalCmds, expectedSequence)

	// Wait for door open duration plus a buffer and verify the hall call is cleared.
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	if elevState.Requests[floorButtonRequest][elevconsts.HallUp] != 0 {
		t.Errorf("Expected hall call for floor %v to be cleared", floorButtonRequest)
	}
}

// Test Full Journey
func TestFullJourney(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := 0
	floorButtonRequest := 2
	floorButtonRequest2 := 3
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	stopClearingChannel := make(chan bool)
	go func(chan<- bool) {
		for {
			select {
			case <-stopClearingChannel:
				return
			case <-commandChannel: //Keep Clearing
			}
		}
	}(stopClearingChannel)

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	stopClearingChannel <- true

	// hall call down
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorButtonRequest, Button: elevconsts.HallDown}}
	time.Sleep(TEST_DELAY)
	if elevState.Requests[floorButtonRequest][elevconsts.HallDown] != 1 {
		t.Errorf("Expected hall call for floor %v (HallDown)", floorButtonRequest)
	}

	// Simulate arrival at floor
	time.Sleep(TEST_DELAY)
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: 2}}
	time.Sleep(TEST_DELAY)
	hallCmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, hallCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.FloorIndicatorCommand{Floor: floorButtonRequest}},
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}},
		{Value: elevcmd.DoorOpenCommand{}},
	})
	if elevState.Requests[floorButtonRequest][elevconsts.HallDown] != 0 {
		t.Errorf("Expected hall call for floor %v to be cleared", floorButtonRequest)
	}
	doorCloseCmds := waitForCommands(commandChannel, elevState.doorOpenDuration+TEST_DELAY)
	checkOrder(t, doorCloseCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.DoorCloseCommand{}},
	})

	// cab call for floor 3
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.ButtonPressEvent{Floor: floorButtonRequest2, Button: elevconsts.Cab}}
	time.Sleep(TEST_DELAY)
	if elevState.Requests[floorButtonRequest2][elevconsts.Cab] != 1 {
		t.Errorf("Expected cab call for floor %v to be registered", floorButtonRequest2)
	}
	UpMotorDirectionCmd := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, UpMotorDirectionCmd, []elevcmd.ElevatorCommand{
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Up}},
	})

	// Simulate arriving at floor 3.
	eventChannel <- elevevent.ElevatorEvent{Value: elevevent.FloorSensorEvent{Floor: floorButtonRequest2}}
	time.Sleep(TEST_DELAY)
	cabCmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, cabCmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.FloorIndicatorCommand{Floor: floorButtonRequest2}},
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Stop}},
		{Value: elevcmd.DoorOpenCommand{}},
	})
	time.Sleep(elevState.doorOpenDuration + TEST_DELAY)
	if elevState.Requests[floorButtonRequest2][elevconsts.Cab] != 0 {
		t.Errorf("Expected cab call for floor %v to be cleared", floorButtonRequest2)
	}
	if elevState.Floor != floorButtonRequest2 {
		t.Errorf("Expected elevator to finish at floor %v, got %d", floorButtonRequest2, elevState.Floor)
	}
}

func TestInitialisationBetweenFloors(t *testing.T) {
	_ = logger.GetLoggerConfigured(zerolog.Disabled)
	eventChannel := make(chan elevevent.ElevatorEvent, 100)
	commandChannel := make(chan elevcmd.ElevatorCommand, 100)
	clearUpDownOnArrival := false

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	elevState := NewElevatorState(eventChannel, commandChannel, clearUpDownOnArrival)

	floorStart := -1
	go func() {
		time.Sleep(TEST_DELAY)
		eventChannel <- elevevent.ElevatorEvent{Value: elevevent.RequestFloorEvent{Floor: floorStart}}
	}()

	err := elevState.Start(ctx, wg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	cmds := waitForCommands(commandChannel, TEST_DELAY)
	checkOrder(t, cmds, []elevcmd.ElevatorCommand{
		{Value: elevcmd.MotorDirCommand{Dir: elevconsts.Down}},
	})
	if elevState.Dirn != elevconsts.Down {
		t.Errorf("Expected elevator Dirn to be Down, got %v", elevState.Dirn)
	}
	if elevState.Behaviour != elevconsts.Moving {
		t.Errorf("Expected elevator Behaviour to be Moving, got %v", elevState.Behaviour)
	}
}
