package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
)

var Log = logger.GetLogger()

type ElevatorMessage struct {
	ElevatorData     elevmetadata.ElevMetaData
	ElevatorState    elevstate.ElevatorState
	RequestStatesMsg requestconfirmation.RequestArrayMessage
}

func MakeElevatorMessage(
	meta *elevmetadata.ElevMetaData,
	state *elevstate.ElevatorState,
) ElevatorMessage {
	return ElevatorMessage{
		ElevatorData:  *meta,
		ElevatorState: *state,
	}
}

type ElevNetBroadcast struct {
	broadcasting       bool                       // internal variable
	startStopCh        chan int                   // internal variable
	conn               net.PacketConn             // changed type from *net.UDPConn
	broadCastingPeriod time.Duration              // internal variable
	metaData           *elevmetadata.ElevMetaData // internal variable
	elevatorState      *elevstate.ElevatorState

	stateOutChannel         <-chan elevstate.ElevatorState
	outboundReqArrayChannel <-chan requestconfirmation.RequestArrayMessage
}

func NewElevNetBroadcast(metaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState, outboundReqCh <-chan requestconfirmation.RequestArrayMessage) *ElevNetBroadcast {
	return &ElevNetBroadcast{
		broadcasting:            false,
		startStopCh:             make(chan int),
		metaData:                metaData,
		elevatorState:           elevatorState,
		stateOutChannel:         stateOutChannel,
		outboundReqArrayChannel: outboundReqCh,
	}
}

func (enb *ElevNetBroadcast) Start(broadcastPeriod time.Duration) error {
	if enb.broadcasting {
		return errors.New("nodeBroadcast is already broadcasting")
	}
	if enb.metaData == nil {
		return errors.New("metaData is nil")
	}
	enb.broadCastingPeriod = broadcastPeriod

	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		fmt.Println("Error: Socket:", err)
	}
	syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		fmt.Println("Error: SetSockOpt REUSEADDR:", err)
	}
	syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	if err != nil {
		fmt.Println("Error: SetSockOpt BROADCAST:", err)
	}
	syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
	if err != nil {
		fmt.Println("Error: SetSockOpt REUSEPORT:", err)
	}
	syscall.Bind(s, &syscall.SockaddrInet4{Port: 9999})
	if err != nil {
		fmt.Println("Error: Bind:", err)
	}

	f := os.NewFile(uintptr(s), "")
	c, err := net.FilePacketConn(f)
	if err != nil {
		fmt.Println("Error: FilePacketConn:", err)
	}
	f.Close()
	enb.conn = c

	udpAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	// Use DialUDP and store the connection as a net.PacketConn.
	// conn, err := net.DialUDP("udp", nil, udpAddr)
	// if err != nil {
	// 	return fmt.Errorf("error creating UDP Socket: %v", err)
	// }
	// conn.SetWriteBuffer(BUFFER_LENGTH)
	// enb.conn = conn

	go func() {
		timeTicker := time.NewTicker(enb.broadCastingPeriod)
		defer timeTicker.Stop()
		defer enb.conn.Close()
		enb.broadcasting = true

		var latestState elevstate.ElevatorState
		var latestRequests requestconfirmation.RequestArrayMessage

		for {
			select {
			case updatedState, ok := <-enb.stateOutChannel:
				if !ok {
					return
				}
				// Cache the updated state.
				latestState = updatedState

			case updatedRequests, ok := <-enb.outboundReqArrayChannel:
				if !ok {
					return
				}
				latestRequests = updatedRequests

			case <-timeTicker.C:
				// On each tick, send out the most recent state.
				msg := ElevatorMessage{
					ElevatorData:     *enb.metaData,
					ElevatorState:    latestState,
					RequestStatesMsg: latestRequests,
				}
				jsonData, err := json.Marshal(msg)
				if err != nil {
					Log.Error().Msgf("Error marshalling JSON: %v", err)
					continue
				}
				// Use WriteTo to send to the remote address.
				_, err = enb.conn.WriteTo(jsonData, udpAddr)
				if err != nil {
					Log.Error().Msgf("Error writing to UDP Socket: %v", err)
					continue
				}
				Log.Debug().Msgf("Sent Packet: %v", string(jsonData))

			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Broadcasting...")
					return
				}
			}
		}
	}()

	Log.Info().Msgf("Started To Broadcast")

	return nil
}

func (enb *ElevNetBroadcast) Stop() error {
	if !enb.broadcasting {
		return errors.New("cannot stop broadcasting if NodeBroadcast is not broadcasting")
	}

	enb.startStopCh <- 0
	enb.broadcasting = false

	return nil
}
