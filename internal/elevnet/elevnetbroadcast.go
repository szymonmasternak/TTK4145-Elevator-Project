package elevnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

// Constants for timing.
const (
	ACK_PERIOD               = 500 * time.Millisecond
	RETRANSMISSION_THRESHOLD = 1000 * time.Millisecond
	MAX_RETRIES              = 5
	//BUFFER_LENGTH            = 1024
)

// AckMessage carries an acknowledgment, including when it was sent.
type AckMessage struct {
	ID           int
	Acknowledged bool
	TimeSent     time.Time
}

// ElevatorMessage carries the elevator state plus its ack info.
type ElevatorMessage struct {
	AckMsg        AckMessage
	ElevatorData  elevmetadata.ElevMetaData
	ElevatorState elevstate.ElevatorState
}

func MakeAckMessage(id int, ack bool) AckMessage {
	// This function is used by the broadcaster when constructing a non-ACK message.
	// (The listener will create a fresh ACK with time.Now().)
	return AckMessage{
		ID:           id,
		Acknowledged: ack,
		TimeSent:     time.Now(),
	}
}

func MakeElevatorMessage(meta *elevmetadata.ElevMetaData, state *elevstate.ElevatorState, id int, ack bool) ElevatorMessage {
	return ElevatorMessage{
		ElevatorData:  *meta,
		ElevatorState: *state,
		AckMsg:        MakeAckMessage(id, ack),
	}
}

// ElevNetBroadcast encapsulates the broadcast logic.
type ElevNetBroadcast struct {
	broadcasting       bool
	startStopCh        chan int
	conn               *net.UDPConn
	broadCastingPeriod time.Duration
	metaData           *elevmetadata.ElevMetaData
	elevatorState      *elevstate.ElevatorState

	stateInChannel  <-chan elevstate.ElevatorState
	stateOutChannel <-chan elevstate.ElevatorState

	// ackChan is still used as a fallback, but we now use an ackRegistry.
	ackChan chan AckMessage

	// ackRegistry maps message IDs to channels waiting for ACKs.
	ackRegistry map[int]chan AckMessage
	registryMu  sync.Mutex

	msgQueue chan ElevatorMessage
}

func NewElevNetBroadcast(metaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateInChannel <-chan elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState) *ElevNetBroadcast {
	return &ElevNetBroadcast{
		broadcasting:    false,
		startStopCh:     make(chan int),
		metaData:        metaData,
		elevatorState:   elevatorState,
		stateInChannel:  stateInChannel,
		stateOutChannel: stateOutChannel,
		ackChan:         make(chan AckMessage, 10000),
		ackRegistry:     make(map[int]chan AckMessage),
		msgQueue:        make(chan ElevatorMessage, 100),
	}
}

// waitForAck registers a dedicated channel for the given messageID and returns it.
func (enb *ElevNetBroadcast) waitForAck(messageID int) <-chan AckMessage {
	ch := make(chan AckMessage, 1)
	enb.registryMu.Lock()
	enb.ackRegistry[messageID] = ch
	enb.registryMu.Unlock()
	return ch
}

// deliverAck looks up the waiting channel for messageID and sends the ack.
func (enb *ElevNetBroadcast) deliverAck(ack AckMessage) {
	enb.registryMu.Lock()
	if ch, ok := enb.ackRegistry[ack.ID]; ok {
		Log.Debug().Msgf("Delivering ACK for message ID %d with timestamp %s", ack.ID, ack.TimeSent.Format(time.RFC3339Nano))
		ch <- ack
		delete(enb.ackRegistry, ack.ID)
	} else {
		Log.Debug().Msgf("No waiting channel for ACK with message ID %d", ack.ID)
	}
	enb.registryMu.Unlock()
}

// Start begins the broadcast process.
func (enb *ElevNetBroadcast) Start(broadcastPeriod time.Duration) error {
	if enb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}
	if enb.metaData == nil {
		return errors.New("metaData is nil")
	}
	enb.broadCastingPeriod = broadcastPeriod

	udpAddress, err := net.ResolveUDPAddr("udp", "10.100.23.255:9999")
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	enb.conn, err = net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return fmt.Errorf("error creating UDP Socket: %v", err)
	}
	enb.conn.SetWriteBuffer(BUFFER_LENGTH)

	enb.broadcasting = true

	// 1. Producer Goroutine.
	go func() {
		timeTicker := time.NewTicker(enb.broadCastingPeriod)
		defer timeTicker.Stop()
		index := 0
		var latestState elevstate.ElevatorState

		for {
			select {
			case updatedState, ok := <-enb.stateOutChannel:
				if !ok {
					return
				}
				latestState = updatedState
			case <-timeTicker.C:
				msg := ElevatorMessage{
					ElevatorData:  *enb.metaData,
					ElevatorState: latestState,
					AckMsg:        MakeAckMessage(index, false), // non-ACK message
				}
				enb.msgQueue <- msg
				index++
			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping message production...")
					return
				}
			}
		}
	}()

	// 2. Sender Goroutine.
	go func() {
		for {
			select {
			case msg := <-enb.msgQueue:
				jsonData, err := json.Marshal(msg)
				if err != nil {
					Log.Error().Msgf("Error marshalling JSON: %v", err)
					continue
				}
				_, err = enb.conn.Write(jsonData)
				if err != nil {
					Log.Error().Msgf("Error writing to UDP Socket: %v", err)
					continue
				}
				Log.Debug().Msgf("Sent Packet: %v", string(jsonData))

				ackCh := enb.waitForAck(msg.AckMsg.ID)
				retries := 0
				for {
					ctx, cancel := context.WithTimeout(context.Background(), RETRANSMISSION_THRESHOLD)
					select {
					case ack := <-ackCh:
						if ack.TimeSent.IsZero() {
							Log.Error().Msgf("Received ACK with zero timestamp for message ID %d", ack.ID)
							// We treat this as a timeout.
						}
						rtt := time.Since(ack.TimeSent)
						if rtt > ACK_PERIOD {
							Log.Warn().Msgf("Delayed ACK for message ID %d: RTT %v; retransmitting", ack.ID, rtt)
							msg.AckMsg.TimeSent = time.Now()
							enb.msgQueue <- msg
						} else {
							Log.Debug().Msgf("ACK received for message ID %d in %v", ack.ID, rtt)
						}
						cancel()
						break
					case <-ctx.Done():
						cancel()
						retries++
						if retries >= MAX_RETRIES {
							Log.Error().Msgf("Max retries reached for message ID %d; giving up", msg.AckMsg.ID)
							break
						}
						Log.Warn().Msgf("Timeout waiting for ACK for message ID %d; retransmitting (retry %d)", msg.AckMsg.ID, retries)
						msg.AckMsg.TimeSent = time.Now()
						enb.msgQueue <- msg
						ackCh = enb.waitForAck(msg.AckMsg.ID)
					}
					break
				}
			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping sender goroutine...")
					return
				}
			}
		}
	}()

	// 3. ACK Listener Goroutine.
	go func() {
		ackBuffer := make([]byte, BUFFER_LENGTH)
		for {
			n, _, err := enb.conn.ReadFromUDP(ackBuffer)
			if err != nil {
				Log.Error().Msgf("Error reading from UDP: %v", err)
				continue
			}
			// Unmarshal into an ElevatorMessage, then extract the AckMsg.
			var elevMsg ElevatorMessage
			if err := json.Unmarshal(ackBuffer[:n], &elevMsg); err != nil {
				Log.Error().Msgf("Error unmarshalling ElevatorMessage: %v", err)
				continue
			}
			if elevMsg.ElevatorData.Identifier != enb.metaData.Identifier {
				Log.Debug().Msgf("Ignoring ACK from elevator %s, expected %s",
					elevMsg.ElevatorData.Identifier, enb.metaData.Identifier)
				continue // Skip ACKs from unexpected sources.
			}
			// Deliver the ACK.
			enb.deliverAck(elevMsg.AckMsg)
		}
	}()

	Log.Info().Msg("Started broadcasting")
	return nil
}

func (enb *ElevNetBroadcast) Stop() error {
	if !enb.broadcasting {
		return errors.New("cannot stop broadcasting if nodeBroadcast is not broadcasting")
	}
	enb.startStopCh <- 0
	enb.broadcasting = false
	return nil
}
