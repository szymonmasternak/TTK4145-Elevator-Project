package nodebroadcast

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

type NodeBroadcast struct {
	SoftwareVersion string `json:"software_version"`
	IpAddress       string `json:"ip_address"`
	PortNumber      int    `json:"port_number"`
	NodeNumber      int    `json:"node_number"`
	DeviceType      string `json:"device_type"`

	broadcasting bool         //internal variable
	startStopCh  chan int     //internal variable
	conn         *net.UDPConn //internal variable
}

const (
	BROADCAST_PERIOD = 100 * time.Millisecond
)

var Log = logger.GetLogger()

func NewNodeBroadcast(softwareversion string, ipaddress string, portnumber int, nodenumber int, devicetype string) *NodeBroadcast {
	return &NodeBroadcast{
		SoftwareVersion: softwareversion,
		IpAddress:       ipaddress,
		PortNumber:      portnumber,
		NodeNumber:      nodenumber,
		DeviceType:      devicetype,

		broadcasting: false,
		startStopCh:  make(chan int),
	}
}

func (nb *NodeBroadcast) StartBroadcasting() error {
	if nb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}

	udpAddress, err := net.ResolveUDPAddr("udp", "255.255.255.255:9999")
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return fmt.Errorf("error dialing UDP: %v", err)
	}

	nb.conn, err = net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return fmt.Errorf("error creating UDP Socket: %v", err)
	}
	nb.conn.SetWriteBuffer(1024)

	go func() {
		timeTicker := time.NewTicker(BROADCAST_PERIOD)
		defer timeTicker.Stop()
		defer nb.conn.Close()
		nb.broadcasting = true

		for {
			select {
			case <-timeTicker.C:
				jsonData, err := json.Marshal(nb)
				if err != nil {
					Log.Error().Msgf("Error marshalling JSON: %v", err)
				}
				_, err = conn.Write(jsonData)
				if err != nil {
					Log.Error().Msgf("Error writing to UDP Socket: %v", err)
				}

				Log.Debug().Msgf("Sendt Packet: %v", string(jsonData))

			case val := <-nb.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping periodic task...")
					return
				}
			}
		}
	}()

	Log.Info().Msgf("Node Started To Broadcast")

	return nil
}

func (nb *NodeBroadcast) StopBroadcasting() error {
	if !nb.broadcasting {
		return errors.New("cannot stop broadcasting if NodeBroadcast is not broadcasting")
	}

	nb.startStopCh <- 0
	nb.broadcasting = false

	return nil
}
