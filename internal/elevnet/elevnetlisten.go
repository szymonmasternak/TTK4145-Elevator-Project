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

const ConnectionCheck = 500 * time.Millisecond
const WaitForReconnection = 1000 * time.Millisecond

// ElevatorListObject holds a received message and its status.
type ElevatorListObject struct {
	msg              ElevatorMessage
	timeSeen         time.Time
	disconnected     bool
	timeDisconnected time.Time
}

// ElevNetListen encapsulates the listening functionality.
type ElevNetListen struct {
	// Channel for forwarding received broadcast messages.
	ElevatorsFoundOnNetwork chan ElevatorMessage
	stateInChannel          <-chan elevstate.ElevatorState
	stateOutChannel         <-chan elevstate.ElevatorState

	listening     bool                       // internal flag
	startStopCh   chan int                   // for shutdown signaling
	conn          *net.UDPConn               // UDP connection used for listening
	elevMetaData  *elevmetadata.ElevMetaData // metadata for this elevator
	elevatorArray []ElevatorListObject
	ElevatorState *elevstate.ElevatorState
	ackChan       chan AckMessage // Channel for ACKs
}

func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateInChannel <-chan elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState) *ElevNetListen {
	return &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan ElevatorMessage, 10000), // buffered to avoid blocking
		stateInChannel:          stateInChannel,
		stateOutChannel:         stateOutChannel,
		listening:               false,
		startStopCh:             make(chan int),
		conn:                    nil,
		elevMetaData:            elevMetaData,
		ElevatorState:           elevatorState,
		ackChan:                 make(chan AckMessage, 10000),
	}
}

// Start starts the listener by binding to the UDP address and launching goroutines.
func (enl *ElevNetListen) Start() error {
	localAddr, err := net.ResolveUDPAddr("udp", "10.100.23.255:9999") // or "0.0.0.0:9999"
	if err != nil {
		return fmt.Errorf("error resolving local UDP address: %v", err)
	}
	enl.conn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("error listening on UDP: %v", err)
	}

	listenBuffer := make([]byte, BUFFER_LENGTH)
	enl.listening = true
	Log.Info().Msgf("Started listening")

	go func() {
		for {
			n, _, err := enl.conn.ReadFromUDP(listenBuffer)
			if err != nil {
				// If the connection is closed, exit gracefully.
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					return
				}
				Log.Error().Msgf("Error reading UDP message: %v", err)
				continue
			}
			var msg ElevatorMessage
			err = json.Unmarshal(listenBuffer[:n], &msg)
			if err != nil {
				Log.Error().Msgf("Error deserialising JSON: %v", err)
				continue
			}

		}
	}()

	go func() {
		for {
			select {
			case msg := <-enl.ElevatorsFoundOnNetwork:
				// Call your AddNodeToList() here
				enl.AddNodeToList(msg)
				Log.Info().Msg("I see the list")
			case val := <-enl.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping Listening task...")
					// Clean up, then return
					return
				}
			}
		}
	}()
	go func() {
		defer enl.conn.Close()
		for {
			select {
			case val := <-enl.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping Listening task...")
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
	var elevatorFound bool
	elevatorFound = false
	for i := 0; i < len(nl.elevatorArray); i++ {
		if msg.ElevatorData.Identifier == nl.elevatorArray[i].msg.ElevatorData.Identifier {
			elevatorFound = true
			nl.elevatorArray[i].timeSeen = time.Now()
			nl.elevatorArray[i].disconnected = false
			nl.elevatorArray[i].msg.ElevatorState = msg.ElevatorState
			break
		}
	}
	if !elevatorFound {
		nl.elevatorArray = append(nl.elevatorArray, ElevatorListObject{msg, time.Now(), false, time.Time{}})
	}
	Logger.Info().Msgf("Node list: ")

	filtered := nl.elevatorArray[:0]

	for i := 0; i < len(nl.elevatorArray); i++ {
		if time.Now().Before(nl.elevatorArray[i].timeSeen.Add(ConnectionCheck)) {
			filtered = append(filtered, nl.elevatorArray[i])
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
	nl.elevatorArray = filtered
}
