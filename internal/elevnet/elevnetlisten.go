package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

const ConnectionCheck = -200 * time.Millisecond

type ElevatorListObject struct {
	ElevatorData   elevmetadata.ElevMetaData
	timeSeen       time.Time
	stateInChannel <-chan elevstate.ElevatorState
}

type ElevNetListen struct {
	ElevatorsFoundOnNetwork chan elevmetadata.ElevMetaData // Returns elevators broadcasted on network

	listening      bool                       // Internal variable
	startStopCh    chan int                   // Internal variable
	conn           *net.UDPConn               // Internal variable
	elevMetaData   *elevmetadata.ElevMetaData // Internal variable
	elevatorArray  []ElevatorListObject
	stateInChannel chan elevstate.ElevatorState // ✅ Fixed channel direction (bidirectional)
}

// ✅ Constructor remains unchanged
func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData, stateInChannel chan elevstate.ElevatorState) *ElevNetListen {
	return &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan elevmetadata.ElevMetaData),
		listening:               false,
		startStopCh:             make(chan int),
		conn:                    nil,
		elevMetaData:            elevMetaData,
		elevatorArray:           []ElevatorListObject{},
		stateInChannel:          stateInChannel,
	}
}

// ✅ Listens for UDP messages and forwards them into stateInChannel
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
		defer enl.conn.Close() // ✅ Ensure the connection is closed when the function exits
		for {
			n, _, err := enl.conn.ReadFromUDP(listenBuffer)
			if err != nil {
				if errors.Is(err, net.ErrClosed) { // ✅ Handle closed connection error
					Log.Warn().Msg("UDP connection closed, stopping listener.")
					return
				}
				Log.Error().Msgf("Error reading UDP message: %v", err)
				continue
			}

			var receivedState elevstate.ElevatorState
			err = json.Unmarshal(listenBuffer[:n], &receivedState)
			if err != nil {
				Log.Error().Msgf("Error deserializing Elevator State JSON: %v", err)
				continue
			}

			// ✅ Ignore messages from itself to prevent feedback loops
			if receivedState.MetaData.Identifier == enl.elevMetaData.Identifier {
				continue
			}

			// ✅ Forward valid state updates to `stateInChannel`
			select {
			case enl.stateInChannel <- receivedState:
				Log.Debug().Msgf("Received and forwarded Elevator State update: %+v", receivedState)
			default:
				Log.Warn().Msg("Dropped received state update (stateInChannel full)")
			}
		}
	}()

	go func() {
		for {
			select {
			case val := <-enl.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping Listening task...")
					enl.listening = false
					enl.conn.Close() // ✅ Explicitly close the connection before exiting
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

	// ✅ Check if `enl.conn` is `nil` before closing it
	if enl.conn != nil {
		enl.conn.Close()
		enl.conn = nil // Ensure it's reset
	}

	return nil
}

// ✅ Updates and filters the elevator list dynamically
func (nl *ElevNetListen) AddNodeToList(n elevmetadata.ElevMetaData) {
	var repeat bool
	repeat = false
	for i := 0; i < len(nl.elevatorArray); i++ {
		if n.Identifier == nl.elevatorArray[i].ElevatorData.Identifier {
			repeat = true
			nl.elevatorArray[i].timeSeen = time.Now()
		}
	}
	if !repeat {
		nl.elevatorArray = append(nl.elevatorArray, ElevatorListObject{n, time.Now(), nl.stateInChannel})
	}
	Logger.Info().Msgf("Node list: ")

	filtered := nl.elevatorArray[:0] // Keep only valid elements

	for i := 0; i < len(nl.elevatorArray); i++ {
		if nl.elevatorArray[i].timeSeen.After(time.Now().Add(ConnectionCheck)) {
			filtered = append(filtered, nl.elevatorArray[i]) // Keep only non-stale nodes
			fmt.Printf("%v, ", nl.elevatorArray[i].ElevatorData.Identifier)
		} else {
			Logger.Info().Msg("Node timed out, removing from the list")
		}
	}
	fmt.Printf("\n")
	nl.elevatorArray = filtered // Update original slice
}
