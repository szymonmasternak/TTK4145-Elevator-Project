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

const ConnectionCheck = 200 * time.Millisecond
const WaitForReconnection = 500 * time.Millisecond

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
		ackChan:                 make(chan AckMessage, 100),
	}
}

// Start starts the listener by binding to the UDP address and launching goroutines.
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
	Log.Info().Msgf("Started listening")

	go func() {
		for {
			n, addr, err := enl.conn.ReadFromUDP(listenBuffer)
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

			// If the message is not already an ACK, send an ACK response.
			if !msg.AckMsg.Acknowledged {
				// Create an ACK response with a fresh timestamp.
				ackResponse := ElevatorMessage{
					ElevatorData:  msg.ElevatorData,
					ElevatorState: msg.ElevatorState,
					AckMsg: AckMessage{
						ID:           msg.AckMsg.ID,
						Acknowledged: true,
						TimeSent:     time.Now(), // Set fresh timestamp.
					},
				}
				enl.ackChan <- ackResponse.AckMsg
				jsonAck, err := json.Marshal(ackResponse)
				if err != nil {
					Log.Error().Msgf("Error marshalling ACK JSON: %v", err)
				} else {
					_, err = enl.conn.WriteToUDP(jsonAck, addr)
					if err != nil {
						Log.Error().Msgf("Error writing ACK to UDP Socket: %v", err)
					} else {
						Log.Debug().Msgf("Sent ACK for message ID %d, time stamp: %s", msg.AckMsg.ID, msg.AckMsg.TimeSent.Format(time.RFC3339))
					}
				}

				// Forward the original broadcast message non-blockingly.
				select {
				case enl.ElevatorsFoundOnNetwork <- msg:
				default:
					Log.Warn().Msg("ElevatorsFoundOnNetwork channel full, dropping message")
				}
			} else {
				// If the message is an ACK, forward it to the ack channel non-blockingly.
				select {
				case enl.ackChan <- msg.AckMsg:
					Log.Debug().Msgf("Received ACK for message ID %d", msg.AckMsg.ID)
				default:
					Log.Warn().Msgf("ACK channel full, dropping ACK for message ID %d", msg.AckMsg.ID)
				}
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
