package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/conn"
)

const (
	CONNECTION_CHECK   = 1000 * time.Millisecond
	WAIT_FOR_RECONNECT = 2000 * time.Millisecond
)

type ElevatorListObject struct {
	msg              ElevatorMessage
	timeSeen         time.Time
	disconnected     bool
	timeDisconnected time.Time
}

type ElevNetListen struct {
	ElevatorsFoundOnNetwork chan ElevatorMessage
	stateOutChannel         <-chan elevstate.ElevatorState
	inboundReqArrayChannel  chan<- requestconfirmation.RequestArrayMessage
	alivePeersChannel       chan<- []string
	stateMapChannel         chan<- map[string]elevstate.ElevatorState

	listening         bool
	startStopCh       chan int
	conn              net.PacketConn
	elevatorArrayChan chan func([]ElevatorListObject) []ElevatorListObject
}

func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState, inboundReqArrayCh chan<- requestconfirmation.RequestArrayMessage, alivePeersChannel chan<- []string) *ElevNetListen {
	enl := &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan ElevatorMessage, 10000),
		stateOutChannel:         stateOutChannel,
		inboundReqArrayChannel:  inboundReqArrayCh,
		alivePeersChannel:       alivePeersChannel,
		listening:               false,
		startStopCh:             make(chan int),
		elevatorArrayChan:       make(chan func([]ElevatorListObject) []ElevatorListObject),
	}

	go func() {
		elevatorArray := []ElevatorListObject{}
		for fn := range enl.elevatorArrayChan {
			elevatorArray = fn(elevatorArray)
		}
	}()

	return enl
}

func (enl *ElevNetListen) Start() error {
	enl.conn = conn.DialBroadcastUDP(9999)

	listenBuffer := make([]byte, BUFFER_LENGTH)
	enl.listening = true
	Log.Info().Msgf("Started listening on shared port")

	go func() {
		for {
			n, _, err := enl.conn.ReadFrom(listenBuffer[0:])
			if err != nil {
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
			enl.ElevatorsFoundOnNetwork <- msg
			enl.inboundReqArrayChannel <- requestconfirmation.RequestArrayMessage{
				Identifier:   msg.ElevatorData.Identifier,
				RequestArray: msg.RequestStates,
			}
		}
	}()

	go func() {
		for {
			select {
			case msg := <-enl.ElevatorsFoundOnNetwork:
				enl.AddNodeToList(msg)
				enl.BroadcastAlivePeers()
			case val := <-enl.startStopCh:
				if val == 0 {
					Log.Info().Msg("Stopping Listening task...")
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

func (enl *ElevNetListen) AddNodeToList(msg ElevatorMessage) {
	enl.elevatorArrayChan <- func(array []ElevatorListObject) []ElevatorListObject {
		var elevatorFound bool
		for i := 0; i < len(array); i++ {
			if msg.ElevatorData.Identifier == array[i].msg.ElevatorData.Identifier {
				elevatorFound = true
				array[i].timeSeen = time.Now()
				array[i].disconnected = false
				array[i].msg.ElevatorState = msg.ElevatorState
				break
			}
		}
		if !elevatorFound {
			array = append(array, ElevatorListObject{msg, time.Now(), false, time.Time{}})
		}

		filtered := array[:0]
		for i := 0; i < len(array); i++ {
			if time.Now().Before(array[i].timeSeen.Add(CONNECTION_CHECK)) {
				filtered = append(filtered, array[i])
				fmt.Printf("%v, ", array[i].msg.ElevatorData.Identifier)
			} else {
				array[i].disconnected = true
				if array[i].timeDisconnected.IsZero() {
					array[i].timeDisconnected = time.Now()
				}
				Log.Info().Msg("Elevator disconnected, waiting for reconnect")
				if time.Now().Before(array[i].timeDisconnected.Add(WAIT_FOR_RECONNECT)) {
					filtered = append(filtered, array[i])
				} else {
					Log.Info().Msg("Elevator didn't reconnect in time, removing from list")
				}
			}
		}
		Log.Info().Msgf("Current list of elevators: %v", func() []string {
			var ids []string
			for _, e := range filtered {
				ids = append(ids, e.msg.ElevatorData.Identifier)
			}
			return ids
		}())
		fmt.Printf("\n")
		return filtered
	}
}

func (enl *ElevNetListen) GetElevatorStateMap() map[string]elevstate.ElevatorState {
	result := make(chan map[string]elevstate.ElevatorState)
	enl.elevatorArrayChan <- func(array []ElevatorListObject) []ElevatorListObject {
		states := make(map[string]elevstate.ElevatorState)
		for _, obj := range array {
			states[obj.msg.ElevatorData.Identifier] = obj.msg.ElevatorState
		}
		go func() { result <- states }()
		return array
	}
	return <-result
}

func (enl *ElevNetListen) GetAliveElevatorIDs() []string {
	result := make(chan []string)
	enl.elevatorArrayChan <- func(array []ElevatorListObject) []ElevatorListObject {
		var ids []string
		for _, obj := range array {
			if !obj.disconnected {
				ids = append(ids, obj.msg.ElevatorData.Identifier)
			}
		}
		go func() { result <- ids }()
		return array
	}
	return <-result
}

func (enl *ElevNetListen) BroadcastAlivePeers() {
	if enl.alivePeersChannel == nil {
		return
	}
	select {
	case enl.alivePeersChannel <- enl.GetAliveElevatorIDs():
	default:
		Log.Warn().Msg("alivePeersChannel full, skipping broadcast")
	}
}

func (enl *ElevNetListen) BroadcastStateMap() {
	if enl.stateMapChannel == nil {
		return
	}
	select {
	case enl.stateMapChannel <- enl.GetElevatorStateMap():
	default:
		Log.Warn().Msg("stateMapChannel full, skipping broadcast")
	}
}
