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
	ACK_PERIOD               = 50 * time.Millisecond
	RETRANSMISSION_THRESHOLD = 100 * time.Millisecond
	MAX_RETRIES              = 5
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
		ackChan:         make(chan AckMessage, 100),
		ackRegistry:     make(map[int]chan AckMessage),
		msgQueue:        make(chan ElevatorMessage, 100),
	}
}

// waitForAck registers a dedicated channel for the given messageID and returns it.
// The sender goroutine will wait on this channel.
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
		// Deliver the ACK and remove the entry.
		ch <- ack
		delete(enb.ackRegistry, ack.ID)
	} else {
		Log.Debug().Msgf("No waiting channel for ACK with message ID %d", ack.ID)
	}
	enb.registryMu.Unlock()
}

// Start begins the broadcast process.
// It launches three goroutines:
// 1. A producer that enqueues messages based on ticker events and state updates.
// 2. A sender that dequeues messages, sends them, and waits for ACKs (with retries).
// 3. An ACK listener that reads ACK messages from the UDP connection and delivers them.
func (enb *ElevNetBroadcast) Start(broadcastPeriod time.Duration) error {
	if enb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}
	if enb.metaData == nil {
		return errors.New("metaData is nil")
	}
	enb.broadCastingPeriod = broadcastPeriod

	udpAddress, err := net.ResolveUDPAddr("udp", enb.metaData.GetIPAddressPort())
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
					AckMsg:        MakeAckMessage(index, false),
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
				// Serialize and send the message.
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

				// Register and wait for the ACK.
				ackCh := enb.waitForAck(msg.AckMsg.ID)
				retries := 0
				// Use a loop so that if we need to retransmit, we can wait again.
				for {
					// Create a context with timeout for waiting.
					ctx, cancel := context.WithTimeout(context.Background(), RETRANSMISSION_THRESHOLD)
					select {
					case ack := <-ackCh:
						rtt := time.Since(ack.TimeSent)
						if rtt > ACK_PERIOD {
							Log.Warn().Msgf("Delayed ACK for message ID %d: RTT %v; retransmitting", ack.ID, rtt)
							// Retransmit: update the timestamp and requeue.
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
						// Re-register for the ACK since the previous channel might have been used.
						ackCh = enb.waitForAck(msg.AckMsg.ID)
					}
					// Exit the inner loop if an ACK has been received in time.
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
			var ack AckMessage
			if err := json.Unmarshal(ackBuffer[:n], &ack); err != nil {
				Log.Error().Msgf("Error unmarshalling ACK: %v", err)
				continue
			}
			// Deliver the ACK using the registry.
			enb.deliverAck(ack)
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
