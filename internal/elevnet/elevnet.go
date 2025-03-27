package elevnet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Logger = logger.GetLogger()

// All structs for network messages
type NetworkPacketType int

const (
	PACKET_TYPE_HEARTBEAT NetworkPacketType = iota
	PACKET_TYPE_STATE
	PACKET_TYPE_BUTTON
	PACKET_TYPE_ACK // Simple acknowledgment
)

type NetworkPacket struct {
	PacketType NetworkPacketType         `json:"packet_type"`
	MetaData   elevmetadata.ElevMetaData `json:"meta_data"`
	Time       time.Time                 `json:"time"`
}

type HeartBeatPacket struct {
	NetworkPacket
}

type StatePacket struct {
	NetworkPacket
	State elevstate.ElevatorState `json:"state"`
}

type ButtonPacket struct {
	NetworkPacket
	Floor  int               `json:"floor"`
	Button elevconsts.Button `json:"button"`
}

type AckPacket struct {
	NetworkPacket
	MessageHash string `json:"message_hash"` // Hash of the original message
}

// From here is all the network stuff

const (
	// ticker intervals
	HEARTBEAT_INTERVAL   = 500 * time.Millisecond
	SEND_STATE_INTERVAL  = 1 * time.Second
	CHECK_NODES_INTERVAL = 1 * time.Second

	// timeout values
	NODE_TIMEOUT            = 3 * time.Second
	ACKNOWLEDGEMENT_TIMEOUT = 200 * time.Millisecond

	// max acknowledgement retries
	MAX_RETRIES = 3

	// channel capacity
	CHANNEL_SIZE = 10

	// buffers
	BUFFER_SIZE_HEARTBEAT  = 1024
	BUFFER_SIZE_DIRECT_MSG = 2048

	// broadcast addresses
	BROADCAST_RX_ADDRESS = "224.0.0.1" //https://gist.github.com/fiorix/9664255
	BROADCAST_TX_ADDRESS = "255.255.255.255"
)

type Node struct {
	MetaData          elevmetadata.ElevMetaData
	State             elevstate.ElevatorState
	LastHeartbeatTime time.Time
	LastStateUpdate   time.Time
	Alive             bool
	Mutex             sync.Mutex
}

type ElevatorNetwork struct {
	metaData *elevmetadata.ElevMetaData

	nodes      map[string]*Node
	nodesMutex sync.Mutex

	localState   *elevstate.ElevatorState
	statePointer sync.Mutex

	broadcastConn   *net.UDPConn
	broadcastTXAddr *net.UDPAddr

	udpConn *net.UDPConn

	receiveChannel chan AckPacket
	initialised    bool
}

// generic decode helper
func decodePacket[T any](data []byte, name string) (T, error) {
	var out T
	err := json.Unmarshal(data, &out)
	if err != nil {
		Logger.Error().Msgf("Failed to decode %s: %v", name, err)
	}
	return out, err
}

func NewElevatorNetwork(metaData *elevmetadata.ElevMetaData, state *elevstate.ElevatorState) *ElevatorNetwork {
	return &ElevatorNetwork{
		metaData:       metaData,
		nodes:          make(map[string]*Node, CHANNEL_SIZE),
		localState:     state,
		receiveChannel: make(chan AckPacket, CHANNEL_SIZE),
		initialised:    true,
	}
}

func (en *ElevatorNetwork) Start(ctx context.Context, wg *sync.WaitGroup) error {
	//Broadcast UDP Connection
	broadcastRXAddressPort := fmt.Sprintf("%s:%d", BROADCAST_RX_ADDRESS, en.metaData.UdpPort)
	broadcastRXAddress, err := net.ResolveUDPAddr("udp", broadcastRXAddressPort)
	if err != nil {
		return fmt.Errorf("Error failed to resolve UDP address: %v", err)
	}
	en.broadcastConn, err = net.ListenMulticastUDP("udp", nil, broadcastRXAddress)
	if err != nil {
		return fmt.Errorf("Error failed to listen UDP: %v", err)
	}
	broadcastTXAddressPort := fmt.Sprintf("%s:%d", BROADCAST_TX_ADDRESS, en.metaData.UdpPort)
	en.broadcastTXAddr, err = net.ResolveUDPAddr("udp", broadcastTXAddressPort)
	if err != nil {
		return fmt.Errorf("Failed to resolve broadcast tx address: %v", err)
	}

	//Direct Node->Node UDP Socket
	udpAddressPort := fmt.Sprintf(":%d", en.metaData.PortNumber)
	udpAddress, err := net.ResolveUDPAddr("udp", udpAddressPort)
	if err != nil {
		return fmt.Errorf("error resolving direct address: %v", err)
	}

	en.udpConn, err = net.ListenUDP("udp", udpAddress)
	if err != nil {
		return fmt.Errorf("error setting up direct communication socket: %v", err)
	}

	wg.Add(4)
	//Heartbeat Broadcast
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(HEARTBEAT_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				en.sendElevatorHeartbeat()
			}
		}
	}()

	//Heartbeat Receive Thread
	go func() {
		defer wg.Done()
		buffer := make([]byte, BUFFER_SIZE_HEARTBEAT)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, addr, err := en.broadcastConn.ReadFromUDP(buffer)
				if err != nil {
					Logger.Error().Msgf("Error reading from broadcast: %v", err)
					continue
				}
				message := buffer[0:n]

				var heartBeatPacket HeartBeatPacket
				if err := json.Unmarshal(message, &heartBeatPacket); err != nil {
					Logger.Error().Msgf("Error unmarshalling heartbeat: %v from %s:%d", err, addr.IP.String(), addr.AddrPort().Port())
					continue
				}

				en.handleElevatorHeartbeat(heartBeatPacket)
			}
		}
	}()

	//Receive Direct Messages Thread
	go func() {
		defer wg.Done()
		buffer := make([]byte, BUFFER_SIZE_DIRECT_MSG)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, addr, err := en.udpConn.ReadFromUDP(buffer)
				if err != nil {
					Logger.Error().Msgf("Error reading from direct socket: %v", err)
					continue
				}

				message := buffer[0:n]
				en.handleDirectMessage(message, addr)
			}
		}
	}()

	//Check Nodes & Send State Updates
	go func() {
		defer wg.Done()
		tickerCheckNodes := time.NewTicker(CHECK_NODES_INTERVAL)
		defer tickerCheckNodes.Stop()
		tickerSendStates := time.NewTicker(SEND_STATE_INTERVAL)
		defer tickerSendStates.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tickerCheckNodes.C:
				en.checkNodes()
			case <-tickerSendStates.C:
				en.sendState()
			}
		}
	}()

	return nil
}

func (en *ElevatorNetwork) calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[0:32])
}

func (en *ElevatorNetwork) sendElevatorHeartbeat() {
	heartBeatMessage := HeartBeatPacket{
		NetworkPacket: NetworkPacket{
			PacketType: PACKET_TYPE_HEARTBEAT,
			MetaData:   *en.metaData,
			Time:       time.Now(),
		},
	}

	data, err := json.Marshal(heartBeatMessage)
	if err != nil {
		Logger.Error().Msgf("Failed to serialise heartbeat message: %v", err)
		return
	}

	_, err = en.broadcastConn.WriteToUDP(data, en.broadcastTXAddr)
	if err != nil {
		Logger.Error().Msgf("Failed to broadcast heartbeat: %v", err)
	}
}
func (en *ElevatorNetwork) handleDirectMessage(data []byte, address *net.UDPAddr) {
	var packet NetworkPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		Logger.Error().Msgf("Failed to unmarshal packet: %v", err)
		return
	}

	if packet.MetaData.Identifier == en.metaData.Identifier {
		return
	}

	packetHash := en.calculateHash(data)

	switch packet.PacketType {
	case PACKET_TYPE_STATE:
		statePacket, err := decodePacket[StatePacket](data, "StatePacket")
		if err != nil {
			return
		}
		en.sendAcknowledgement(packetHash, packet.MetaData.Identifier, address)
		en.handleStatePacket(statePacket)

	case PACKET_TYPE_BUTTON:
		buttonPacket, err := decodePacket[ButtonPacket](data, "ButtonPacket")
		if err != nil {
			return
		}
		en.sendAcknowledgement(packetHash, packet.MetaData.Identifier, address)
		en.handleButtonPacket(buttonPacket)

	case PACKET_TYPE_ACK:
		ackPacket, err := decodePacket[AckPacket](data, "AckPacket")
		if err != nil {
			return
		}
		select {
		case en.receiveChannel <- ackPacket:
		default:
			Logger.Error().Msg("Receive channel full: acknowledgment dropped. Increase buffer?")
		}
	default:
		Logger.Warn().Msgf("Received unknown packet type: %v", packet.PacketType)
	}
}

func (en *ElevatorNetwork) handleElevatorHeartbeat(heartbeat HeartBeatPacket) {
	// Ignore our own heartbeats
	if heartbeat.MetaData.Identifier == en.metaData.Identifier {
		Logger.Debug().Msgf("Received own heartbeat")
		return
	}

	Logger.Debug().Msgf("Received heartbeat from %s", heartbeat.MetaData.Identifier)

	en.nodesMutex.Lock()
	defer en.nodesMutex.Unlock()

	nodeID := heartbeat.MetaData.Identifier
	node, exists := en.nodes[nodeID]

	if exists {
		node.Mutex.Lock()
		node.LastHeartbeatTime = time.Now()
		node.Alive = true
		node.Mutex.Unlock()
	} else {
		//node doesnt exist. register new one
		Logger.Info().Msgf("New elevator found: %s at %s:%d", nodeID, heartbeat.MetaData.IpAddress, heartbeat.MetaData.PortNumber)

		//todo perhaps request new states from node?
		en.nodes[nodeID] = &Node{
			MetaData: heartbeat.MetaData,
			State: elevstate.ElevatorState{
				Floor:     -1,
				Dirn:      elevconsts.STOP,
				Behaviour: elevconsts.IDLE,
			},
			LastHeartbeatTime: time.Now(),
			LastStateUpdate:   time.Now(),
			Alive:             true,
		}
	}
}

func (en *ElevatorNetwork) sendAcknowledgement(messageHash string, nodeID string, address *net.UDPAddr) {
	ackPacket := AckPacket{
		NetworkPacket: NetworkPacket{
			PacketType: PACKET_TYPE_ACK,
			MetaData:   *en.metaData,
			Time:       time.Now(),
		},
		MessageHash: messageHash,
	}

	data, err := json.Marshal(ackPacket)
	if err != nil {
		Logger.Error().Msgf("Failed to serialise acknowledge packet %v", err)
		return
	}

	_, err = en.udpConn.WriteToUDP(data, address)
	if err != nil {
		Logger.Error().Msgf("Failed to send acknowledge packet %v", err)
	}
}

func (en *ElevatorNetwork) sendWithRetry(data []byte, node *Node) bool {
	// Extract all needed fields under lock
	node.Mutex.Lock()
	alive := node.Alive
	nodeID := node.MetaData.Identifier
	ip := node.MetaData.IpAddress
	port := node.MetaData.PortNumber
	node.Mutex.Unlock()

	if !alive {
		return false
	}

	nodeAddrStr := fmt.Sprintf("%s:%d", ip, port)
	nodeAddr, err := net.ResolveUDPAddr("udp", nodeAddrStr)
	if err != nil {
		Logger.Error().Msgf("Failed to resolve node address: %v", err)
		return false
	}

	messageHash := en.calculateHash(data)

	for i := 0; i < MAX_RETRIES; i++ {
		_, err := en.udpConn.WriteToUDP(data, nodeAddr)
		if err != nil {
			Logger.Error().Msgf("Failed to send to %s: %v", nodeID, err)
			return false
		}

		select {
		case ack := <-en.receiveChannel:
			if ack.MessageHash == messageHash && ack.MetaData.Identifier == nodeID {
				return true
			}
		case <-time.After(ACKNOWLEDGEMENT_TIMEOUT):
			Logger.Debug().Msgf("Retry sending to %s attempt %d", nodeID, i)
		}
	}

	Logger.Warn().Msgf("Failed to get acknowledgment from %s after %d retries", nodeID, MAX_RETRIES)

	en.nodesMutex.Lock()
	if node, ok := en.nodes[nodeID]; ok {
		node.Mutex.Lock()
		node.Alive = false
		node.Mutex.Unlock()
	}
	en.nodesMutex.Unlock()

	return false
}

func (en *ElevatorNetwork) sendState() {
	en.statePointer.Lock()
	if en.localState == nil {
		en.statePointer.Unlock()
		return
	}
	stateLocal := *en.localState
	en.statePointer.Unlock()

	// Copy alive nodes outside locks
	en.nodesMutex.Lock()
	nodes := make([]*Node, 0, len(en.nodes))
	for _, node := range en.nodes {
		node.Mutex.Lock()
		alive := node.Alive
		node.Mutex.Unlock()
		if alive {
			nodes = append(nodes, node)
		}
	}
	en.nodesMutex.Unlock()

	// Broadcast state
	for _, node := range nodes {
		stateUpdateMsg := StatePacket{
			NetworkPacket: NetworkPacket{
				PacketType: PACKET_TYPE_STATE,
				MetaData:   *en.metaData,
				Time:       time.Now(),
			},
			State: stateLocal,
		}

		data, err := json.Marshal(stateUpdateMsg)
		if err != nil {
			Logger.Error().Msgf("Failed to serialise state update: %v", err)
			continue
		}

		go en.sendWithRetry(data, node)
	}
}

func (en *ElevatorNetwork) handleButtonPacket(packet ButtonPacket) {
	Logger.Error().Msgf("Un implemented TODO function")
}

func (en *ElevatorNetwork) handleStatePacket(packet StatePacket) {
	Logger.Info().Msgf("Received state update from %s", packet.MetaData.Identifier)

	en.nodesMutex.Lock()
	defer en.nodesMutex.Unlock()

	node, exists := en.nodes[packet.MetaData.Identifier]
	if exists {
		node.Mutex.Lock()
		node.State = packet.State
		node.LastStateUpdate = time.Now()
		node.Mutex.Unlock()

		Logger.Debug().Msgf("Updated state for elevator %s", packet.MetaData.Identifier)
		return
	}
	Logger.Error().Msgf("Received state from unknown node: %s", packet.MetaData.Identifier)
}

func (en *ElevatorNetwork) checkNodes() {
	en.nodesMutex.Lock()
	nodeCopies := make(map[string]*Node)
	for id, node := range en.nodes {
		nodeCopies[id] = node
	}
	en.nodesMutex.Unlock()

	now := time.Now()
	for id, node := range nodeCopies {
		node.Mutex.Lock()
		timeSince := now.Sub(node.LastHeartbeatTime)
		if timeSince > NODE_TIMEOUT && node.Alive {
			Logger.Warn().Msgf("Marking Node %s as dead after %v", id, timeSince)
			node.Alive = false
		}
		node.Mutex.Unlock()
	}
}

func (en *ElevatorNetwork) GetNodeStates() map[string]*Node {
	return en.nodes
}

func (en *ElevatorNetwork) GetNodesConnected() int {
	en.nodesMutex.Lock()
	nodes := make([]*Node, 0, len(en.nodes))
	for _, node := range en.nodes {
		nodes = append(nodes, node)
	}
	en.nodesMutex.Unlock()

	count := 0
	for _, node := range nodes {
		node.Mutex.Lock()
		if node.Alive {
			count++
		}
		node.Mutex.Unlock()
	}
	return count
}
