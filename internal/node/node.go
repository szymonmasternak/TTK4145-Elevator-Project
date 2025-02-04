package node

import (
	"encoding/json"
	"fmt"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger() //valid across all files in node folder

const (
	BUFFER_LENGTH = 1024 //for receiving and transmitting
)

type Node struct {
	SoftwareVersion string `json:"software_version"`
	IpAddress       string `json:"ip_address"`
	PortNumber      int    `json:"port_number"`
	NodeNumber      int    `json:"node_number"`
	DeviceType      string `json:"device_type"`
}

func (node *Node) String() string {
	jsonData, err := json.Marshal(node)

	if err != nil {
		Log.Error().Msg("Error Serialing Node Object to JSON")
		return ""
	}
	return string(jsonData)
}

func (node *Node) GetIPAddressPort() string {
	return fmt.Sprintf("%s:%d", node.IpAddress, node.PortNumber)
}
