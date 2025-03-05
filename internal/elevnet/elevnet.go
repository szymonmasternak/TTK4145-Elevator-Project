package elevnet

import (
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

func NewElevatorNetwork(elevMeta *elevmetadata.ElevMetaData, elevState *elevstate.ElevatorState, stateInChannel <-chan elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState) *ElevatorNetwork {
	return &ElevatorNetwork{
		Broadcast: NewElevNetBroadcast(elevMeta, elevState, stateInChannel, stateOutChannel),
		Listen:    NewElevNetListen(elevMeta, elevState, stateInChannel, stateOutChannel),
	}
}
