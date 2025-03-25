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

	listening        bool                       // internal flag
	startStopCh      chan int                   // for shutdown signaling
	conn             *net.UDPConn               // UDP connection used for listening
	elevMetaData     *elevmetadata.ElevMetaData // metadata for this elevator
	elevatorArray    []ElevatorListObject
	elevatorArrayMtx sync.Mutex
	ElevatorState    *elevstate.ElevatorState
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
	}
}

// Start starts the listener by binding to the UDP address and launching goroutines.
func (enl *ElevNetListen) Start() error {
	localAddr, err := net.ResolveUDPAddr("udp", "10.22.138.8:9999")
	if err != nil {
		return fmt.Errorf("error resolving local UDP address: %v", err)
	}

	// Create UDP socket manually with SO_REUSEADDR
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		return fmt.Errorf("socket error: %v", err)
	}
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("setsockopt SO_REUSEADDR error: %v", err)
	}

	sockaddr := &syscall.SockaddrInet4{Port: localAddr.Port}
	copy(sockaddr.Addr[:], net.IPv4zero.To4())
	if err := syscall.Bind(fd, sockaddr); err != nil {
		return fmt.Errorf("bind error: %v", err)
	}

	// Convert to *net.UDPConn
	file := os.NewFile(uintptr(fd), "udp-reuseaddr")
	c, err := net.FilePacketConn(file)
	if err != nil {
		return fmt.Errorf("FilePacketConn error: %v", err)
	}
	file.Close()
	conn, ok := c.(*net.UDPConn)
	if !ok {
		return fmt.Errorf("failed to cast to UDPConn")
	}
	enl.conn = conn

	listenBuffer := make([]byte, BUFFER_LENGTH)
	enl.listening = true
	Log.Info().Msgf("Started listening on shared port")

	go func() {
		for {
			n, _, err := enl.conn.ReadFromUDP(listenBuffer)
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
		}
	}()

	go func() {
		for {
			select {
			case msg := <-enl.ElevatorsFoundOnNetwork:
				enl.AddNodeToList(msg)
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
	var elavatorFound bool
	elavatorFound = false
	for i := 0; i < len(nl.elevatorArray); i++ {
		if msg.ElevatorData.Identifier == nl.elevatorArray[i].msg.ElevatorData.Identifier {
			elavatorFound = true
			nl.elevatorArray[i].timeSeen = time.Now()
			nl.elevatorArray[i].disconnected = false
			nl.elevatorArray[i].msg.ElevatorState = msg.ElevatorState
			break
		}
	}
	if !elavatorFound {
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
