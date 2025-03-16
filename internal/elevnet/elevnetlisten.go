package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
	"sync"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

const ConnectionCheck = 200 * time.Millisecond
const WaitForReconnection = 500 * time.Millisecond

type ElevatorListObject struct {
	msg              ElevatorMessage
	timeSeen         time.Time
	disconnected     bool
	timeDisconnected time.Time
}

type ElevNetListen struct {
	ElevatorsFoundOnNetwork chan ElevatorMessage //returns elevators broadcasted on network
	stateInChannel          <-chan elevstate.ElevatorState
	stateOutChannel         <-chan elevstate.ElevatorState

	listening     bool                       //internal variable
	startStopCh   chan int                   //internal variable
	conn          *net.UDPConn               //internal variable
	ElevMetaData  *elevmetadata.ElevMetaData //internal variable
	mu            sync.Mutex
	elevatorArray []ElevatorListObject
	ElevatorState *elevstate.ElevatorState
}

func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateInChannel <-chan elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState) *ElevNetListen {
	return &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan ElevatorMessage),
		stateInChannel:          stateInChannel,
		stateOutChannel:         stateOutChannel,
		listening:               false,
		startStopCh:             make(chan int),
		conn:                    nil,
		ElevMetaData:            elevMetaData,
		ElevatorState:           elevatorState,
	}
}

func (enl *ElevNetListen) Start() error {
	udpAddress, err := net.ResolveUDPAddr("udp", enl.ElevMetaData.GetIPAddressPort())
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
			// var node elevmetadata.ElevMetaData
			// err = json.Unmarshal(listenBuffer[:n], &node)
			var msg ElevatorMessage
			err = json.Unmarshal(listenBuffer[:n], &msg)

			if err != nil {
				Log.Error().Msgf("Error deserialising JSON: %v", err)
			} else {
				enl.ElevatorsFoundOnNetwork <- msg
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

func (nl *ElevNetListen) AddNodeToList(msg ElevatorMessage) {
	var elavatorFound bool
	elavatorFound = false
	for i := 0; i < len(nl.elevatorArray); i++ {
		if msg.ElevatorData.Identifier == nl.elevatorArray[i].msg.ElevatorData.Identifier {
			elavatorFound = true
			nl.elevatorArray[i].timeSeen = time.Now()
			nl.elevatorArray[i].disconnected = false
			nl.elevatorArray[i].msg.ElevatorState = msg.ElevatorState
			break
		}
	}
	if !elavatorFound {
		nl.elevatorArray = append(nl.elevatorArray, ElevatorListObject{msg, time.Now(), false, time.Time{}})
	}
	Logger.Info().Msgf("Node list: ")

	filtered := nl.elevatorArray[:0] // Keep only valid elements

	for i := 0; i < len(nl.elevatorArray); i++ {
		if time.Now().Before(nl.elevatorArray[i].timeSeen.Add(ConnectionCheck)) {
			filtered = append(filtered, nl.elevatorArray[i]) // Keep only non-stale nodes
			fmt.Printf("%v, ", nl.elevatorArray[i].msg.ElevatorData.Identifier)
		} else {
			nl.elevatorArray[i].disconnected = true
			if nl.elevatorArray[i].timeDisconnected.IsZero() {
				nl.elevatorArray[i].timeDisconnected = time.Now()
			}
			Logger.Info().Msg("Elevator disconnected, waiting for reconnect")
			if time.Now().Before(nl.elevatorArray[i].timeDisconnected.Add(WaitForReconnection)) {
				filtered = append(filtered, nl.elevatorArray[i])
			} else {
				Logger.Info().Msg("Elevator didn't reconnect in time, removing from list")
			}
		}
	}
	fmt.Printf("\n")
	nl.elevatorArray = filtered // Update original slice
}

func (nl *ElevNetListen) GetElevatorMessageMap() map[string]ElevatorMessage {
    nl.mu.Lock()
    defer nl.mu.Unlock()

    messages := make(map[string]ElevatorMessage)
    for _, obj := range nl.elevatorArray {
        identifier := obj.msg.ElevatorData.Identifier
        messages[identifier] = obj.msg
    }
    return messages
}
