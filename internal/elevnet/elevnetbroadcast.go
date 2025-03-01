package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

type ElevNetBroadcast struct {
	broadcasting       bool                       //internal variable
	startStopCh        chan int                   //internal variable
	conn               *net.UDPConn               //internal variable
	broadCastingPeriod time.Duration              //internal variable
	metaData           *elevmetadata.ElevMetaData //internal variable
}

func NewElevNetBroadcast(metaData *elevmetadata.ElevMetaData) *ElevNetBroadcast {
	return &ElevNetBroadcast{
		broadcasting: false,
		startStopCh:  make(chan int),
		metaData:     metaData,
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
				jsonData, err := json.Marshal(enb.metaData)
				if err != nil {
					Log.Error().Msgf("Error marshalling JSON: %v", err)
				}
				_, err = enb.conn.Write(jsonData)
				if err != nil {
					Log.Error().Msgf("Error writing to UDP Socket: %v", err)
				}

				Log.Debug().Msgf("Sent Packet: %v", string(jsonData))

			case val := <-enb.startStopCh:
				if val == 0 {
					Log.Info().Msgf("Stopping Broadcasting task...")
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
