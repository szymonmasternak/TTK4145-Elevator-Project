package elevnet

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Logger = logger.GetLogger()

const (
	BUFFER_LENGTH = 1024 //for receiving and transmitting
)

type ElevatorNetwork struct {
	Broadcast *ElevNetBroadcast
	Listen    *ElevNetListen
}

func NewElevatorNetwork(elevMeta *elevmetadata.ElevMetaData, stateOutChannel <-chan elevstate.ElevatorState, stateInChannel chan elevstate.ElevatorState) *ElevatorNetwork {
	broadcast := NewElevNetBroadcast(elevMeta, stateOutChannel)
	listen := NewElevNetListen(elevMeta, stateInChannel)

	if broadcast == nil || listen == nil {
		Logger.Error().Msg("Failed to initialize network components")
		return nil
	}

	network := &ElevatorNetwork{
		Broadcast: broadcast,
		Listen:    listen,
	}

	// ✅ Start broadcasting and listening
	err := network.Broadcast.Start(100 * time.Millisecond)
	if err != nil {
		Logger.Error().Msgf("Failed to start broadcasting: %v", err)
		return nil
	}

	err = network.Listen.Start()
	if err != nil {
		Logger.Error().Msgf("Failed to start listening: %v", err)
		return nil
	}

	return network
}
