package main

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

func main() {
	var Log = logger.GetLogger() //valid across all files in node folder
	Log.Info().Msg("Started!")

	Elevator := elevator.NewElevator("")
	Elevator.Start()

	// Input poll rate
	// inputPollRate := 25 * time.Millisecond

	// // Initialize elevalgorithm
	// elevalgorithm.DriverInit("localhost:15657", 4)
	// elevalgorithm.FSMinit()

	// // Channels for inputs
	// drv_buttons := make(chan elevalgorithm.ButtonEvent)
	// drv_floors := make(chan int)
	// drv_stop := make(chan bool)
	// drv_obstr := make(chan bool)

	// // Start polling input devices
	// go elevalgorithm.PollButtons(drv_buttons)
	// go elevalgorithm.PollFloorSensor(drv_floors)
	// go elevalgorithm.PollStopButton(drv_stop)
	// go elevalgorithm.PollObstructionSwitch(drv_obstr)

	// // Elevator loop
	// ticker := time.NewTicker(inputPollRate)
	// defer ticker.Stop()

	// for {
	// 	select {
	// 	case buttonPress := <-drv_buttons:
	// 		elevalgorithm.Fsm_onRequestButtonPress(buttonPress.Floor, elevalgorithm.Button(buttonPress.Button))

	// 	case floorArrival := <-drv_floors:
	// 		elevalgorithm.Fsm_onFloorArrival(floorArrival)

	// 	case <-ticker.C: // This checks the timer periodically
	// 		if elevalgorithm.Timer_timedOut() {
	// 			if !elevalgorithm.GetObstruction() {
	// 				elevalgorithm.Timer_stop()
	// 				elevalgorithm.Fsm_onDoorTimeout()
	// 			}

	// 		}
	// 	}
	// }
}
