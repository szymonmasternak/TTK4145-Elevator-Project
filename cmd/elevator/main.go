package main

import (
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
	elev.Start()
	for {
	}
}
