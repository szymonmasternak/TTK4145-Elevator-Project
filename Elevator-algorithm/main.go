package main

import (
	"elevator-algorithm/elevator"
	"elevator-algorithm/elevio"
	"fmt"
	"time"
)

func main() {
	fmt.Println("Started!")

	// Input poll rate
	inputPollRate := 25 * time.Millisecond

	// Initialize elevator
	elevio.Init("localhost:15657", 4)
	elevator.Fsm_onInitBetweenFloors()

	// Channels for inputs
	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_stop := make(chan bool)
	drv_obstr := make(chan bool)

	// Start polling input devices
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollStopButton(drv_stop)
	go elevio.PollObstructionSwitch(drv_obstr)

	// Elevator loop
	ticker := time.NewTicker(inputPollRate)
	defer ticker.Stop()

	for {
		select {
		case buttonPress := <-drv_buttons:
			elevator.Fsm_onRequestButtonPress(buttonPress.Floor, elevator.Button(buttonPress.Button))

		case floorArrival := <-drv_floors:
			elevator.Fsm_onFloorArrival(floorArrival)

		case <-ticker.C: // This properly checks the timer periodically
			if elevator.Timer_timedOut() {
				if !elevio.GetObstruction() {
					fmt.Println(elevio.GetObstruction())
					elevator.Timer_stop()
					fmt.Println("Timer timed out!")
					elevator.Fsm_onDoorTimeout()
				}

			}
		}
	}
}
