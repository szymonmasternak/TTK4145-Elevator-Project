package elevmetadata

import (
	"encoding/json"
	"fmt"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type ElevMetaData struct {
	SoftwareVersion string `json:"software_version"`
	IpAddress       string `json:"ip_address"`
	PortNumber      uint16 `json:"port_number"`
	Identifier      string `json:"identifier"`
}

func (elevMetaData *ElevMetaData) String() string {
	jsonData, err := json.Marshal(elevMetaData)

	if err != nil {
		Log.Error().Msg("Error Serialising ElevMetaData Object to JSON")
		return ""
	}
	return string(jsonData)
}

func (elevMetaData *ElevMetaData) GetIPAddressPort() string {
	return fmt.Sprintf("%s:%d", elevMetaData.IpAddress, elevMetaData.PortNumber)
}
