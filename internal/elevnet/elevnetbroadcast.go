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

// Constants
const (
	ACK_PERIOD               = 400 * time.Millisecond
	RETRANSMISSION_THRESHOLD = 700 * time.Millisecond
	MAX_RETRIES              = 5
	//BUFFER_LENGTH            = 65535 // or however large you'd like
)

// Use a well-known multicast group and port.
// 224.0.0.x is within the local administrative scope.
//const MULTICAST_ADDRESS = "224.0.0.1:9999"

type AckMessage struct {
	ID           int
	Acknowledged bool
	TimeSent     time.Time
}

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

// ElevNetBroadcast handles sending elevator state and receiving ACKs.
type ElevNetBroadcast struct {
	broadcasting       bool
	startStopCh        chan int
	conn               *net.UDPConn
	broadCastingPeriod time.Duration
	metaData           *elevmetadata.ElevMetaData
	elevatorState      *elevstate.ElevatorState

	stateInChannel  <-chan elevstate.ElevatorState
	stateOutChannel <-chan elevstate.ElevatorState

	ackChan     chan AckMessage
	ackRegistry map[int]chan AckMessage
	registryMu  sync.Mutex

	msgQueue chan ElevatorMessage
}

func NewElevNetBroadcast(
	metaData *elevmetadata.ElevMetaData,
	elevatorState *elevstate.ElevatorState,
	stateInChannel <-chan elevstate.ElevatorState,
	stateOutChannel <-chan elevstate.ElevatorState,
) *ElevNetBroadcast {
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

func (enb *ElevNetBroadcast) waitForAck(messageID int) <-chan AckMessage {
	ch := make(chan AckMessage, 1)
	enb.registryMu.Lock()
	enb.ackRegistry[messageID] = ch
	enb.registryMu.Unlock()
	return ch
}

func (enb *ElevNetBroadcast) deliverAck(ack AckMessage) {
	enb.registryMu.Lock()
	if ch, ok := enb.ackRegistry[ack.ID]; ok {
		Log.Debug().Msgf("Delivering ACK for message ID %d", ack.ID)
		ch <- ack
		delete(enb.ackRegistry, ack.ID)
	} else {
		Log.Debug().Msgf("No waiting channel for ACK message ID %d", ack.ID)
	}
	enb.registryMu.Unlock()
}

// Start: now binds to a multicast group so multiple processes on the same
// machine/network can share the same port (9999) and all receive each other's packets.
func (enb *ElevNetBroadcast) Start(broadcastPeriod time.Duration) error {
	if enb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}
	if enb.metaData == nil {
		return errors.New("metaData is nil")
	}
	enb.broadCastingPeriod = broadcastPeriod

	// 1. Listen on the multicast address
	groupAddr, err := net.ResolveUDPAddr("udp", MULTICAST_ADDRESS)
	if err != nil {
		return fmt.Errorf("error resolving multicast address: %v", err)
	}
	enb.conn, err = net.ListenMulticastUDP("udp", nil, groupAddr)
	if err != nil {
		return fmt.Errorf("error creating multicast socket: %v", err)
	}
	if err := enb.conn.SetReadBuffer(BUFFER_LENGTH); err != nil {
		return fmt.Errorf("error setting read buffer: %v", err)
	}

	enb.broadcasting = true

	// 2. Producer goroutine: pushes ElevatorMessages onto msgQueue periodically.
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
				// Create a new non-ACK message
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

	// 3. Sender goroutine: sends the queued messages to the multicast address, then waits for ACK.
	go func() {
		for {
			select {
			case msg := <-enb.msgQueue:
				// Marshal the message
				jsonData, err := json.Marshal(msg)
				if err != nil {
					Log.Error().Msgf("Error marshalling JSON: %v", err)
					continue
				}

				// Send to the same multicast group
				_, err = enb.conn.WriteToUDP(jsonData, groupAddr)
				if err != nil {
					Log.Error().Msgf("Error writing to multicast socket: %v", err)
					continue
				}
				Log.Debug().Msgf("Sent Packet: %s", string(jsonData))

				// Wait for ACK
				ackCh := enb.waitForAck(msg.AckMsg.ID)
				retries := 0
			RETRY_LOOP:
				for {
					ctx, cancel := context.WithTimeout(context.Background(), RETRANSMISSION_THRESHOLD)
					select {
					case ack := <-ackCh:
						// If we get an ACK from ourselves or from the correct ID, handle it
						rtt := time.Since(ack.TimeSent)
						Log.Debug().Msgf("ACK received for message ID %d in %v", ack.ID, rtt)
						cancel()
						break RETRY_LOOP

					case <-ctx.Done():
						cancel()
						retries++
						if retries >= MAX_RETRIES {
							Log.Error().Msgf("Max retries reached for message ID %d; giving up", msg.AckMsg.ID)
							break RETRY_LOOP
						}
						Log.Warn().Msgf("Timeout waiting for ACK for message ID %d; retransmitting (retry %d)", msg.AckMsg.ID, retries)
						_, err = enb.conn.WriteToUDP(jsonData, groupAddr)
						if err != nil {
							Log.Error().Msgf("Error retransmitting to multicast: %v", err)
							break RETRY_LOOP
						}
						ackCh = enb.waitForAck(msg.AckMsg.ID)
					}
				}

			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping sender goroutine...")
					return
				}
			}
		}
	}()

	// 4. ACK listener goroutine: receives messages from the same multicast socket
	//    and delivers them if they're ACKs for *our* message ID + ElevatorData.Identifier.
	go func() {
		ackBuffer := make([]byte, BUFFER_LENGTH)
		for {
			n, _, err := enb.conn.ReadFromUDP(ackBuffer)
			if err != nil {
				Log.Error().Msgf("Error reading from UDP: %v", err)
				continue
			}
			var elevMsg ElevatorMessage
			if err := json.Unmarshal(ackBuffer[:n], &elevMsg); err != nil {
				Log.Error().Msgf("Error unmarshalling ElevatorMessage: %v", err)
				continue
			}

			// If it's an ACK, check if it's meant for our ID.
			if elevMsg.AckMsg.Acknowledged {
				// Optional: filter by "elevMsg.ElevatorData.Identifier == enb.metaData.Identifier"
				// if you only want to handle ACKs intended specifically for us.
				enb.deliverAck(elevMsg.AckMsg)
			}
		}
	}()

	Log.Info().Msg("Started broadcasting via multicast")
	return nil
}

// Stop ends all the goroutines and closes the connection.
func (enb *ElevNetBroadcast) Stop() error {
	if !enb.broadcasting {
		return errors.New("cannot stop broadcasting if nodeBroadcast is not broadcasting")
	}
	enb.startStopCh <- 0
	enb.broadcasting = false
	return enb.conn.Close()
}
