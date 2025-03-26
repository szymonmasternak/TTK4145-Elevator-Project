package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/conn"
)

var Log = logger.GetLogger()

type ElevatorMessage struct {
	ElevatorData  elevmetadata.ElevMetaData
	ElevatorState elevstate.ElevatorState
	RequestStates requestconfirmation.RequestArray
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
	broadcasting            bool                       // internal variable
	startStopCh             chan int                   // internal variable
	conn                    net.PacketConn             // changed type from *net.UDPConn
	broadCastingPeriod      time.Duration              // internal variable
	metaData                *elevmetadata.ElevMetaData // internal variable
	stateOutChannel         <-chan elevstate.ElevatorState
	outboundReqArrayChannel <-chan requestconfirmation.RequestArrayMessage
}

func NewElevNetBroadcast(metaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState, outboundReqCh <-chan requestconfirmation.RequestArrayMessage) *ElevNetBroadcast {
	return &ElevNetBroadcast{
		broadcasting:            false,
		startStopCh:             make(chan int),
		metaData:                metaData,
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

	conn := conn.DialBroadcastUDP(9999)
	enb.conn = conn

	udpAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
	if err != nil {
		return fmt.Errorf("error resolving UDP Address: %v", err)
	}

	go func() {
		timeTicker := time.NewTicker(enb.broadCastingPeriod)
		defer timeTicker.Stop()
		defer enb.conn.Close()
		enb.broadcasting = true

		var latestState elevstate.ElevatorState
		var latestRequests requestconfirmation.RequestArray

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
				latestRequests = updatedRequests.RequestArray

			case <-timeTicker.C:
				// On each tick, send out the most recent state.
				msg := ElevatorMessage{
					ElevatorData:  *enb.metaData,
					ElevatorState: latestState,
					RequestStates: latestRequests,
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
