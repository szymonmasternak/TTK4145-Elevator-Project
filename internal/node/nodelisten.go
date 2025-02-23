package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Logger = logger.GetLogger()

const ConnectionCheck = -200 * time.Millisecond

type LastSeenNode struct {
	Node     Node
	timeSeen time.Time
}

type NodeListen struct {
	Node //inherits struct from node package

	NodesFoundOnNetwork chan Node //returns node broadcasted on network

	listening   bool           //internal variable
	startStopCh chan int       //internal variable
	conn        *net.UDPConn   //internal variable
	nodeArray   []LastSeenNode //list of nodes??
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
	udpAddress, err := net.ResolveUDPAddr("udp", nl.GetIPAddressPort())
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

func (nl *NodeListen) AddNodeToList(n Node) {
	var repeat bool
	repeat = false
	for i := 0; i < len(nl.nodeArray); i++ {
		if n.DeviceType == nl.nodeArray[i].Node.DeviceType {
			repeat = true
			nl.nodeArray[i].timeSeen = time.Now()
		}
	}
	if !repeat {
		nl.nodeArray = append(nl.nodeArray, LastSeenNode{n, time.Now()})
	}
	Logger.Info().Msgf("Node list: ")

	filtered := nl.nodeArray[:0] // Keep only valid elements

	for i := 0; i < len(nl.nodeArray); i++ {
		if nl.nodeArray[i].timeSeen.After(time.Now().Add(ConnectionCheck)) {
			filtered = append(filtered, nl.nodeArray[i]) // Keep only non-stale nodes
			fmt.Printf("%v, ", nl.nodeArray[i].Node.DeviceType)
		} else {
			Logger.Info().Msg("Node timed out, removing from the list")
		}
	}
	fmt.Printf("\n")
	nl.nodeArray = filtered // Update original slice
}
