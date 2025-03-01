package elevnet

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
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

func NewElevatorNetwork(elevMeta *elevmetadata.ElevMetaData) *ElevatorNetwork {
	return &ElevatorNetwork{
		Broadcast: NewElevNetBroadcast(elevMeta),
		Listen:    NewElevNetListen(elevMeta),
	}
}
