package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
)

type ElevNetListen struct {
	ElevatorsFoundOnNetwork chan elevmetadata.ElevMetaData //returns elevators broadcasted on network

	listening    bool                       //internal variable
	startStopCh  chan int                   //internal variable
	conn         *net.UDPConn               //internal variable
	elevMetaData *elevmetadata.ElevMetaData //internal variable
}

func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData) *ElevNetListen {
	return &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan elevmetadata.ElevMetaData),
		listening:               false,
		startStopCh:             make(chan int),
		conn:                    nil,
		elevMetaData:            elevMetaData,
	}
}

func (enl *ElevNetListen) Start() error {
	udpAddress, err := net.ResolveUDPAddr("udp", enl.elevMetaData.GetIPAddressPort())
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	enl.conn, err = net.ListenUDP("udp", udpAddress)
	if err != nil {
		return fmt.Errorf("error creating UDP Socket: %v", err)
	}
	listenBuffer := make([]byte, BUFFER_LENGTH)
	enl.listening = true

	go func() {
		for {
			n, _, err := enl.conn.ReadFromUDP(listenBuffer)
			if err != nil {
				Log.Error().Msgf("Error reading UDP message: %v", err)
				continue
			}
			var node elevmetadata.ElevMetaData
			err = json.Unmarshal(listenBuffer[:n], &node)
			if err != nil {
				Log.Error().Msgf("Error deserialising JSON: %v", err)
			} else {
				enl.ElevatorsFoundOnNetwork <- node
			}
		}
	}()

	go func() {
		defer enl.conn.Close()
		for {
			select {
			case val := <-enl.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Listening task...")
					return
				}
			}
		}
	}()

	return nil
}

func (enl *ElevNetListen) Stop() error {
	if !enl.listening {
		return errors.New("cannot stop listening if nodeListen is not listening")
	}

	enl.startStopCh <- 0
	enl.listening = false

	return nil
}
