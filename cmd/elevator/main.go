package main

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"

	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"
)

var Logger = logger.GetLogger()

func main() {
	// Starting Programme
	Logger.Info().Msg("Starting Elevator Programme")

	identifier := elevutils.ProcessCmdArgs()

	elev := elevator.NewElevator(identifier)

	Logger.Info().Msgf("Elevator: %v", elev.MetaData.String())

	elev.Network.Broadcast.Start(time.Millisecond * 1000)
	elev.Network.Listen.Start()

	for {
		select {
		case elevatorFound := <-elev.Network.Listen.ElevatorsFoundOnNetwork:
			Logger.Info().Msgf("Elevator found on network: %v", elevatorFound.String())
		}
	}
}
