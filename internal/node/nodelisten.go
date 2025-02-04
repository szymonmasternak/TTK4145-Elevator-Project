package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
)

type NodeListen struct {
	Node //inherits struct from node package

	NodesFoundOnNetwork chan Node //returns node broadcasted on network

	listening   bool         //internal variable
	startStopCh chan int     //internal variable
	conn        *net.UDPConn //internal variable
}

func NewNodeListen(node Node) *NodeListen {
	return &NodeListen{
		Node:                node,
		NodesFoundOnNetwork: make(chan Node),
		listening:           false,
		startStopCh:         make(chan int),
		conn:                nil,
	}
}

func (nl *NodeListen) StartListening() error {
	udpAddress, err := net.ResolveUDPAddr("udp", nl.getIPAddressPort())
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	nl.conn, err = net.ListenUDP("udp", udpAddress)
	if err != nil {
		return fmt.Errorf("error creating UDP Socket: %v", err)
	}
	listenBuffer := make([]byte, BUFFER_LENGTH)
	nl.listening = true

	go func() {
		for {
			n, _, err := nl.conn.ReadFromUDP(listenBuffer)
			if err != nil {
				Log.Error().Msgf("Error reading UDP message: %v", err)
				continue
			}
			var node Node
			err = json.Unmarshal(listenBuffer[:n], &node)
			if err != nil {
				Log.Error().Msgf("Error deserialising JSON: %v", err)
			} else {
				nl.NodesFoundOnNetwork <- node
			}
		}
	}()

	go func() {
		defer nl.conn.Close()
		for {
			select {
			case val := <-nl.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Listening task...")
					return
				}
			}
		}
	}()

	return nil
}

func (nl *NodeListen) StopListening() error {
	if !nl.listening {
		return errors.New("cannot stop listening if nodeListen is not listening")
	}

	nl.startStopCh <- 0
	nl.listening = false

	return nil
}
