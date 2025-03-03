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
)

var Log = logger.GetLogger()

type ElevNetBroadcast struct {
	broadcasting       bool                           // Internal variable
	startStopCh        chan int                       // Internal variable
	conn               *net.UDPConn                   // Internal variable
	broadCastingPeriod time.Duration                  // Internal variable
	metaData           *elevmetadata.ElevMetaData     // Internal variable
	stateOutChannel    <-chan elevstate.ElevatorState // ✅ Changed to a receive-only channel
}

// ✅ Constructor now correctly accepts `stateOutChannel` as receive-only
func NewElevNetBroadcast(metaData *elevmetadata.ElevMetaData, stateOutChannel <-chan elevstate.ElevatorState) *ElevNetBroadcast {
	return &ElevNetBroadcast{
		broadcasting:    false,
		startStopCh:     make(chan int),
		metaData:        metaData,
		stateOutChannel: stateOutChannel, // ✅ Use receive-only channel
	}
}

// ✅ Start broadcasting metadata AND elevator state updates
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

	go func() {
		timeTicker := time.NewTicker(enb.broadCastingPeriod)
		defer timeTicker.Stop()
		defer enb.conn.Close()
		enb.broadcasting = true

		for {
			select {
			case <-timeTicker.C:
				// ✅ Send metadata periodically
				jsonData, err := json.Marshal(enb.metaData)
				if err != nil {
					Log.Error().Msgf("Error marshalling Metadata JSON: %v", err)
					continue
				}
				_, err = enb.conn.Write(jsonData)
				if err != nil {
					Log.Error().Msgf("Error writing Metadata to UDP Socket: %v", err)
				}
				Log.Debug().Msgf("Sent Metadata Packet: %v", string(jsonData))

			case state := <-enb.stateOutChannel:
				jsonState, err := json.Marshal(state)
				if err != nil {
					Log.Error().Msgf("Error marshalling Elevator State JSON: %v", err)
					continue
				}
				_, err = enb.conn.Write(jsonState)
				if err != nil {
					Log.Error().Msgf("Error writing State update to UDP Socket: %v", err)
				}
				// ✅ Add this debug log:
				Log.Info().Msgf("✅ Sent Elevator State Packet: %v", string(jsonState))

			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Broadcasting task...")
					return
				}
			}
		}
	}()

	Log.Info().Msgf("Started Broadcasting...")

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
