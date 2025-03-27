package elevnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
)

const ConnectionCheck = 1000 * time.Millisecond
const WaitForReconnection = 2000 * time.Millisecond

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
	inboundReqArrayChannel  chan<- requestconfirmation.RequestArrayMessage
	alivePeersChannel       chan<- []string

	listening        bool                       // internal flag
	startStopCh      chan int                   // for shutdown signaling
	conn             net.PacketConn             // Changed to use net.PacketConn
	elevMetaData     *elevmetadata.ElevMetaData // metadata for this elevator
	elevatorArray    []ElevatorListObject
	elevatorArrayMtx sync.Mutex
	ElevatorState    *elevstate.ElevatorState
}

func NewElevNetListen(elevMetaData *elevmetadata.ElevMetaData, elevatorState *elevstate.ElevatorState, stateOutChannel <-chan elevstate.ElevatorState, inboundReqArrayCh chan<- requestconfirmation.RequestArrayMessage, alivePeersChannel chan<- []string) *ElevNetListen {
	return &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan ElevatorMessage, 10000), // buffered to avoid blocking
		stateOutChannel:         stateOutChannel,
		inboundReqArrayChannel:  inboundReqArrayCh,
		alivePeersChannel:       alivePeersChannel,

		listening:     false,
		startStopCh:   make(chan int),
		conn:          nil,
		elevMetaData:  elevMetaData,
		ElevatorState: elevatorState,
	}
}

// Start starts the listener by binding to the UDP address and launching goroutines.
func (enl *ElevNetListen) Start() error {
	// localAddr, err := net.ResolveUDPAddr("udp", enl.elevMetaData.IpAddress+":9999")
	// if err != nil {
	// 	return fmt.Errorf("error resolving local UDP address: %v", err)
	// }

	// Create UDP socket manually with SO_REUSEADDR
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
	// Use the returned PacketConn directly.
	enl.conn = c

	listenBuffer := make([]byte, BUFFER_LENGTH)
	enl.listening = true
	Log.Info().Msgf("Started listening on shared port")

	go func() {
		for {
			// Use ReadFrom instead of ReadFromUDP.
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
			if msg.ElevatorData.Identifier != enl.elevMetaData.Identifier {
				enl.inboundReqArrayChannel <- msg.RequestStatesMsg
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

func (nl *ElevNetListen) AddNodeToList(msg ElevatorMessage) {
	nl.elevatorArrayMtx.Lock()
	defer nl.elevatorArrayMtx.Unlock()
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

	filtered := nl.elevatorArray[:0] // Keep only valid elements

	for i := 0; i < len(nl.elevatorArray); i++ {
		if time.Now().Before(nl.elevatorArray[i].timeSeen.Add(ConnectionCheck)) {
			filtered = append(filtered, nl.elevatorArray[i]) // Keep only non-stale nodes
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
	nl.elevatorArray = filtered // Update original slice
}

func (nl *ElevNetListen) GetElevatorStateMap() map[string]elevstate.ElevatorState {
	nl.elevatorArrayMtx.Lock()
	defer nl.elevatorArrayMtx.Unlock()

	messages := make(map[string]elevstate.ElevatorState)
	for _, obj := range nl.elevatorArray {
		identifier := obj.msg.ElevatorData.Identifier
		messages[identifier] = obj.msg.ElevatorState
	}
	return messages
}

func (nl *ElevNetListen) GetAliveElevatorIDs() []string {
	nl.elevatorArrayMtx.Lock()
	defer nl.elevatorArrayMtx.Unlock()

	var aliveIDs []string
	for _, elevator := range nl.elevatorArray {
		if !elevator.disconnected {
			aliveIDs = append(aliveIDs, elevator.msg.ElevatorData.Identifier)
		}
	}
	return aliveIDs
}

func (nl *ElevNetListen) BroadcastAlivePeers() {
	if nl.alivePeersChannel == nil {
		return
	}
	select {
	case nl.alivePeersChannel <- nl.GetAliveElevatorIDs():
	default:
		Logger.Warn().Msg("alivePeersChannel full, skipping broadcast")
	}
}
