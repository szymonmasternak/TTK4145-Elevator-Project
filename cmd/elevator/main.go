package main

import (
	"github.com/rs/zerolog"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"

	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"
)

var Logger = logger.GetLoggerConfigured(zerolog.DebugLevel)

func main() {
	identifier, portNumber, clearUpDownOnArrival, driverIPAddress, udpPort, loadStateFromFile := elevutils.ProcessCmdArgs()

	// Starting Programme
	Logger.Info().Msg("Starting Elevator Programme")

	elev := elevator.NewElevator(identifier, portNumber, driverIPAddress, clearUpDownOnArrival, udpPort, loadStateFromFile)
	elev.Start()

	Logger.Info().Msgf("Elevator: %v", elev.MetaData.String())

	// elev.Network.Broadcast.Start(time.Millisecond * 1000)
	// elev.Network.Listen.Start()

	select {}
}
