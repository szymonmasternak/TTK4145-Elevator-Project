package elevnet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstatenetmsg"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Logger = logger.GetLogger()

// From here is all the network stuff
const (
	HEARTBEAT_INTERVAL              = 500 * time.Millisecond
	SEND_STATE_INTERVAL             = 1 * time.Second
	CHECK_NODES_INTERVAL            = 1 * time.Second
	NODE_TIMEOUT                    = 3 * time.Second
	ACKNOWLEDGEMENT_TIMEOUT         = 200 * time.Millisecond
	MAX_RETRIES                     = 3
	BROADCAST_RX_ADDRESS            = "224.0.0.1" //https://gist.github.com/fiorix/9664255
	BROADCAST_TX_ADDRESS            = "255.255.255.255"
	ELEVNET_NODES_LENGTH            = 10
	ELEVNET_RX_ACK_LENGTH           = 10
	UDP_BUFFER_SIZE                 = 2024
	PRINT_NODES_CONNECTED_INTERNVAL = 1 * time.Second
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

	stateNetChannel chan elevstatenetmsg.ElevatorStateNetMsg
	eventChannel    chan elevevent.ElevatorEvent
}

func NewElevatorNetwork(metaData *elevmetadata.ElevMetaData, state *elevstate.ElevatorState, stateNetChannel chan elevstatenetmsg.ElevatorStateNetMsg, eventChannel chan elevevent.ElevatorEvent) *ElevatorNetwork {
	return &ElevatorNetwork{
		metaData:        metaData,
		nodes:           make(map[string]*Node, ELEVNET_NODES_LENGTH),
		localState:      state,
		receiveChannel:  make(chan AckPacket, ELEVNET_RX_ACK_LENGTH),
		initialised:     true,
		stateNetChannel: stateNetChannel,
		eventChannel:    eventChannel,
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

	wg.Add(6)
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
		buffer := make([]byte, UDP_BUFFER_SIZE)
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
		buffer := make([]byte, UDP_BUFFER_SIZE)
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
		tickerCheckNodesTimeout := time.NewTicker(CHECK_NODES_INTERVAL)
		defer tickerCheckNodesTimeout.Stop()
		tickerSendStates := time.NewTicker(SEND_STATE_INTERVAL)
		defer tickerSendStates.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tickerCheckNodesTimeout.C:
				en.checkNodesTimeout()
			case <-tickerSendStates.C:
				en.sendState()
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-en.stateNetChannel:
				if msg.TimeoutOccured {
					continue
				}

				shouldHandleLocally := en.processHallButtonRequest(msg.Floor, msg.Button)

				en.stateNetChannel <- elevstatenetmsg.ElevatorStateNetMsg{
					Floor:           msg.Floor,
					Button:          msg.Button,
					ShouldDoRequest: shouldHandleLocally,
				}
			}
		}
	}()

	//Debug Thread
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(PRINT_NODES_CONNECTED_INTERNVAL):
				num := en.GetNodesConnected()
				Logger.Info().Msgf("Elevators Connected: %d", num)
			}
		}
	}()

	return nil
}

func (en *ElevatorNetwork) checkNodesTimeout() {
	en.nodesMutex.Lock()

	var offlineNodes []*Node
	var aliveNodes []string

	for id, node := range en.nodes {
		node.Mutex.Lock()
		timeSinceLastNodeHeartBeat := time.Since(node.LastHeartbeatTime)

		if timeSinceLastNodeHeartBeat > NODE_TIMEOUT {
			if node.Alive {
				Logger.Warn().Msgf("Deleting Node %s after not responding for %v", id, timeSinceLastNodeHeartBeat)
				node.Alive = false
				offlineNodes = append(offlineNodes, node)
			}
		} else if node.Alive {
			aliveNodes = append(aliveNodes, id)
		}
		node.Mutex.Unlock()
	}

	aliveNodes = append(aliveNodes, en.metaData.Identifier)
	sort.Strings(aliveNodes)
	en.nodesMutex.Unlock()

	if len(offlineNodes) < 1 {
		return
	}

	if aliveNodes[0] == en.metaData.Identifier {
		Logger.Info().Msgf("We are taking over the hull requests from other nodes")
		en.stealHallRequestsFromNodes(offlineNodes)
	} else {
		Logger.Info().Msgf("Node %s is taking over hull requests from other nodes", aliveNodes[0])
		//We stay quiet
	}
}

func (en *ElevatorNetwork) stealHallRequestsFromNodes(offlineNodes []*Node) {
	for _, node := range offlineNodes {
		node.Mutex.Lock()
		nodeState := node.State
		node.Mutex.Unlock()
		for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
			for btnType := elevconsts.HallUp; btnType <= elevconsts.HallDown; btnType++ {
				if nodeState.Requests[floor][btnType] != 0 {
					Logger.Info().Msgf("Recovering hall call at floor %d, button %s from offline elevator %s", floor, btnType.String(), node.MetaData.Identifier)

					en.eventChannel <- elevevent.ElevatorEvent{
						Value: elevevent.NetworkButtonEvent{
							Floor:  floor,
							Button: elevconsts.Button(btnType),
						},
					}

					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}
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
	//All packets with have NetworkPacket type at the start, parse and identify what packet
	var packet NetworkPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		Logger.Error().Msgf("Failed to unmarshal packet: %v", err)
		return
	}

	//Check if own packet
	if packet.MetaData.Identifier == en.metaData.Identifier {
		return
	}

	packetHash := en.calculateHash(data)

	// Process packet
	switch packet.PacketType {
	case PACKET_TYPE_STATE:
		var statePacket StatePacket
		if err := json.Unmarshal(data, &statePacket); err != nil {
			Logger.Error().Msgf("Failed to unmarshal state packet: %v", err)
			return
		}
		en.sendAcknowledgement(packetHash, packet.MetaData.Identifier, address)
		en.handleStatePacket(statePacket)
	case PACKET_TYPE_ACK:
		var ackPacket AckPacket
		if err := json.Unmarshal(data, &ackPacket); err != nil {
			Logger.Error().Msgf("Failed to unmarshal ACK packet: %v", err)
			return
		}

		select {
		case en.receiveChannel <- ackPacket:
			//Logger.Debug().Msgf("Received acknowledge hash: %s", ackPacket.MessageHash)
		default:
			Logger.Error().Msg("FIX THIS SHOULD NEVER HAPPEN")
		}
	case PACKET_TYPE_DO_REQ:
		var doReqPacket DoRequestPacket
		if err := json.Unmarshal(data, &doReqPacket); err != nil {
			Logger.Error().Msgf("Failed to unmarshal ACK packet: %v", err)
			return
		}
		en.sendAcknowledgement(packetHash, packet.MetaData.Identifier, address)
		en.handleDoReq(doReqPacket)
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

		en.nodes[nodeID] = &Node{
			MetaData: heartbeat.MetaData,
			State: elevstate.ElevatorState{
				Floor:     -1,
				Dirn:      elevconsts.Stop,
				Behaviour: elevconsts.Idle,
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
	node.Mutex.Lock()
	if !node.Alive {
		node.Mutex.Unlock()
		return false
	}
	nodeID := node.MetaData.Identifier
	nodeAddressPort := fmt.Sprintf("%s:%d", node.MetaData.IpAddress, node.MetaData.PortNumber)
	nodeAddr, err := net.ResolveUDPAddr("udp", nodeAddressPort)
	if err != nil {
		Logger.Error().Msgf("Failed to resolve node address: %v", err)
		node.Mutex.Unlock()
		return false
	}
	node.Mutex.Unlock()
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
	node, ok := en.nodes[nodeID]

	wasNodeAlive := false
	if ok {
		//If node is set to alive, then mark it as dead
		node.Mutex.Lock()
		wasNodeAlive = node.Alive
		node.Alive = false
		node.Mutex.Unlock()
	}

	var aliveNodes []string
	for id, n := range en.nodes {
		n.Mutex.Lock()
		if n.Alive {
			aliveNodes = append(aliveNodes, id)
		}
		n.Mutex.Unlock()
	}
	aliveNodes = append(aliveNodes, en.metaData.Identifier)
	sort.Strings(aliveNodes)
	en.nodesMutex.Unlock()

	if wasNodeAlive && aliveNodes[0] == en.metaData.Identifier {
		go func() {
			//stealhallRequests accepts only array of Node object
			nodeList := make([]*Node, 0, 1)
			nodeList = append(nodeList, node)
			en.stealHallRequestsFromNodes(nodeList)
		}()
	}
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

	//Send state to all alive nodes
	en.nodesMutex.Lock()
	nodes := make([]*Node, 0, len(en.nodes))
	for _, node := range en.nodes {
		node.Mutex.Lock()
		if node.Alive {
			nodes = append(nodes, node)
		}
		node.Mutex.Unlock()
	}
	en.nodesMutex.Unlock()

	//Send To Each Node
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

func (en *ElevatorNetwork) GetNodesConnected() int {
	en.nodesMutex.Lock()
	defer en.nodesMutex.Unlock()

	counter := 0

	for _, node := range en.nodes {
		node.Mutex.Lock()
		if node.Alive {
			counter++
		}
		node.Mutex.Unlock()
	}

	return counter
}

func (en *ElevatorNetwork) processHallButtonRequest(floor int, button elevconsts.Button) bool {
	en.statePointer.Lock()
	localCost := en.localState.CalculateTimeToServeReq(floor, button)
	en.statePointer.Unlock()

	bestCost := localCost
	bestElevator := en.metaData.Identifier

	en.nodesMutex.Lock()
	for id, node := range en.nodes {
		node.Mutex.Lock()
		if node.Alive {
			nodeCost := node.State.CalculateTimeToServeReq(floor, button)

			if nodeCost < bestCost {
				bestCost = nodeCost
				bestElevator = id
			}
		}
		node.Mutex.Unlock()
	}
	en.nodesMutex.Unlock()

	if bestElevator != en.metaData.Identifier {
		node, exists := en.nodes[bestElevator]
		if exists {
			doReq := DoRequestPacket{
				NetworkPacket: NetworkPacket{
					PacketType: PACKET_TYPE_DO_REQ,
					MetaData:   *en.metaData,
					Time:       time.Now(),
				},
				Floor:  floor,
				Button: button,
			}
			data, err := json.Marshal(doReq)
			if err != nil {
				Logger.Error().Msgf("Failed to serialise do request: %v", err)
			}
			go en.sendWithRetry(data, node)
			return false
		}
	}
	return true
}

func (en *ElevatorNetwork) handleDoReq(packet DoRequestPacket) {
	Logger.Info().Msgf("Received request to handle hall button at floor %d, button %s", packet.Floor, packet.Button.String())
	en.eventChannel <- elevevent.ElevatorEvent{Value: elevevent.NetworkButtonEvent{Floor: packet.Floor, Button: packet.Button}}
}
