package elevator

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/xyproto/randomstring"
)

var Logger = logger.GetLogger()

type Elevator struct {
	ElevMetaData *elevmetadata.ElevMetaData //this contains all elevator constant metadata

	//ElevIO state

	initialised bool
}

func NewElevator(identifier string) *Elevator {
	if identifier == "" {
		identifier = randomstring.EnglishFrequencyString(10) //this should be random enough
		Logger.Warn().Msgf("No identifier provided, generated random identifier \"%v\"", identifier)
	}

	return &Elevator{
		ElevMetaData: &elevmetadata.ElevMetaData{
			SoftwareVersion: elevutils.GetGitHash(),
			IpAddress:       elevutils.GetLocalIP(),
			PortNumber:      9999,
			Identifier:      identifier,
		},
		initialised: true,
	}
}

func (e *Elevator) Start() {
	// TODO setup all channels?
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
}

func (e *Elevator) Stop() {
	if !e.initialised {
		Logger.Error().Msg("Elevator not initialised")
		return
	}
	// TODO
	// Terminate programme succesfully
	// Send message that programme is stopping
	// Close all sockets
	// etc
}
