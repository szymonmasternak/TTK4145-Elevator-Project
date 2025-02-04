package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

type NodeBroadcast struct {
	Node //inherits struct from node package

	broadcasting       bool          //internal variable
	startStopCh        chan int      //internal variable
	conn               *net.UDPConn  //internal variable
	broadCastingPeriod time.Duration //internal variable
}

func NewNodeBroadcast(node Node, broadcastingPeriod time.Duration) *NodeBroadcast {
	return &NodeBroadcast{
		Node:               node,
		broadcasting:       false,
		startStopCh:        make(chan int),
		broadCastingPeriod: broadcastingPeriod,
	}
}
func (nb *NodeBroadcast) StartBroadcasting() error {
	if nb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}

	udpAddress, err := net.ResolveUDPAddr("udp", nb.GetIPAddressPort())
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	nb.conn, err = net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return fmt.Errorf("error creating UDP Socket: %v", err)
	}
	nb.conn.SetWriteBuffer(BUFFER_LENGTH)

	go func() {
		timeTicker := time.NewTicker(nb.broadCastingPeriod)
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
				_, err = nb.conn.Write(jsonData)
				if err != nil {
					Log.Error().Msgf("Error writing to UDP Socket: %v", err)
				}

				Log.Debug().Msgf("Sent Packet: %v", string(jsonData))

			case val := <-nb.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Broadcasting task...")
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
