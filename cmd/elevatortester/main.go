package main

import (
	"github.com/rs/zerolog"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevator"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Logger = logger.GetLoggerConfigured(zerolog.DebugLevel)

func main() {
	identifier, portNumber, clearAllRequestsAtFloorArrival := elevutils.ProcessCmdArgs()

	// Starting Programme
	Logger.Info().Msg("Starting Elevator Programme")

	elev := elevator.NewElevator(identifier, portNumber, clearAllRequestsAtFloorArrival)
	elev.Start()

	Logger.Info().Msgf("Elevator: %v", elev.MetaData.String())

	select {}
}
