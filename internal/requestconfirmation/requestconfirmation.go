package requestconfirmation

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

var Log = logger.GetLogger()

// RequestState represents the state of a request.
type RequestState int

const (
	REQ_Initial     RequestState = -1
	REQ_None                     = 0
	REQ_Unconfirmed              = 1
	REQ_Confirmed                = 2
	REQ_Completed                = 3
)

// Request holds the state and list of nodes that have confirmed the request.
type Request struct {
	State          RequestState `json:"state"`
	ConsensusPeers []string     `json:"consensus"`
}

// RequestArray is a two-dimensional array of requests.
type RequestArray [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]Request

// RequestConfirmationMap maps a node identifier to its RequestArray.
type RequestConfirmationMap map[string]RequestArray

// RequestMessage is used for local button press or state changes.
type RequestMessage struct {
	Floor  int
	Button elevconsts.Button
	State  RequestState
}

// RequestArrayMessage is used to exchange the entire RequestArray between nodes.
type RequestArrayMessage struct {
	Identifier   string       `json:"id"`
	RequestArray RequestArray `json:"reqArray"`
}

// RequestHandler manages local state and communications with other nodes.
type RequestHandler struct {
	localID string
	// requestMap holds the local state and remote nodes' states.
	requestMap RequestConfirmationMap
	alivePeers []string

	requestUpdateChannel <-chan RequestMessage      // Local messages (e.g., button presses)
	inboundArrayChannel  <-chan RequestArrayMessage // Inbound messages from remote nodes
	outboundArrayChannel chan<- RequestArrayMessage // Outbound messages for broadcasting state
	alivePeersChannel    <-chan []string

	// NEW: localRequestAssignerChannel sends the local RequestArray updates to a request assigner.
	localRequestAssignerChannel chan<- RequestArray

	// lastSent tracks the last broadcasted RequestArray for each peer.
	lastSent RequestConfirmationMap
}

// NewRequestHandler creates a new RequestHandler instance.
// Note the new localRequestAssignerChannel parameter.
func NewRequestHandler(
	localID string,
	requestMsgChannel <-chan RequestMessage,
	inboundArrayChannel <-chan RequestArrayMessage,
	outboundArrayChannel chan<- RequestArrayMessage,
	alivePeersChannel <-chan []string,
	localRequestAssignerChannel chan<- RequestArray,
) *RequestHandler {
	reqMap := NewRequestConfirmationMap(localID)
	lastSent := make(RequestConfirmationMap)
	// Initialize lastSent for the local node.
	lastSent[localID] = reqMap[localID]

	// Broadcast initial state for local node.
	outboundArrayChannel <- RequestArrayMessage{
		Identifier:   localID,
		RequestArray: reqMap[localID],
	}

	return &RequestHandler{
		localID:                     localID,
		requestMap:                  reqMap,
		alivePeers:                  []string{localID},
		requestUpdateChannel:        requestMsgChannel,
		inboundArrayChannel:         inboundArrayChannel,
		outboundArrayChannel:        outboundArrayChannel,
		alivePeersChannel:           alivePeersChannel,
		localRequestAssignerChannel: localRequestAssignerChannel, // NEW
		lastSent:                    lastSent,
	}
}

// NewRequestConfirmationMap initializes the RequestConfirmationMap with the local node.
func NewRequestConfirmationMap(localID string) RequestConfirmationMap {
	reqArr := RequestArray{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			reqArr[floor][btn] = Request{
				State:          REQ_Initial,
				ConsensusPeers: []string{},
			}
		}
	}
	requestMap := make(RequestConfirmationMap)
	requestMap[localID] = reqArr
	Log.Debug().Msgf("Initialized RequestConfirmationMap for %s: %v", localID, requestMap)
	return requestMap
}

// requestArrayEqual compares two RequestArrays.
func requestArrayEqual(a, b RequestArray) bool {
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if a[floor][btn].State != b[floor][btn].State || !stringSliceEqual(a[floor][btn].ConsensusPeers, b[floor][btn].ConsensusPeers) {
				return false
			}
		}
	}
	return true
}

// stringSliceEqual compares two string slices.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// broadcastIfUpdated sends an updated RequestArray for the given peer if it differs from what was last sent.
func (handler *RequestHandler) broadcastIfUpdated(peerID string) {
	current := handler.requestMap[peerID]
	last, exists := handler.lastSent[peerID]
	if !exists || !requestArrayEqual(current, last) {
		handler.outboundArrayChannel <- RequestArrayMessage{
			Identifier:   peerID,
			RequestArray: current,
		}
		if peerID == handler.localID {
			handler.localRequestAssignerChannel <- current
			//Log.Debug().Msgf("Broadcasted local requestStates: %v", current)
		}
		// Update lastSent.
		handler.lastSent[peerID] = current
	}
}

// Start listens for inbound RequestArrayMessages, local RequestMessages,
// and periodically broadcasts updates. It now sends out updated local state
// to both remote nodes and the request assigner.
func (handler *RequestHandler) Start() {
	Log.Debug().Msgf("RequestConfirmer started")
	// Broadcast initial state for local node.
	handler.broadcastIfUpdated(handler.localID)

	confirmationTicker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			// Process inbound messages from remote nodes.
			case incomingMsg := <-handler.inboundArrayChannel:
				if incomingMsg.Identifier == "" {
					Log.Error().Msgf("Empty identifier received")
					break
				}
				Log.Debug().Msgf("Inbound RequestArrayMessage received for %s", incomingMsg.Identifier)
				handler.updateLocalRequestMap(incomingMsg.Identifier, incomingMsg.RequestArray)
				Log.Debug().Msgf("Local map updated: %v", handler.requestMap)
				handler.broadcastIfUpdated(incomingMsg.Identifier)

			// Process local request messages.
			case reqMsg := <-handler.requestUpdateChannel:
				Log.Debug().Msgf("Local RequestMessage received")
				tempArr := handler.requestMap[handler.localID]
				// Update the local RequestArray based on the incoming RequestMessage.
				if tempArr[reqMsg.Floor][reqMsg.Button].State <= REQ_None && reqMsg.State == REQ_Unconfirmed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Unconfirmed
					tempArr[reqMsg.Floor][reqMsg.Button].ConsensusPeers = []string{handler.localID}
				} else if tempArr[reqMsg.Floor][reqMsg.Button].State == REQ_Confirmed && reqMsg.State == REQ_Completed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Completed
					tempArr[reqMsg.Floor][reqMsg.Button].ConsensusPeers = []string{handler.localID}
				}
				handler.requestMap[handler.localID] = tempArr
				if len(handler.alivePeers) == 1 {
					handler.updateLocalRequestMap(handler.localID, handler.requestMap[handler.localID])
				}
				// Broadcast updated state to remote nodes...
				handler.broadcastIfUpdated(handler.localID)
				// And also send the updated local RequestArray to the request assigner.
				Log.Debug().Msgf("Updated local requests: %v", handler.requestMap[handler.localID])

			// Periodic check to broadcast updates if necessary.
			case <-confirmationTicker.C:
				for peerID := range handler.requestMap {
					handler.broadcastIfUpdated(peerID)
				}

			case newPeers := <-handler.alivePeersChannel:
				handler.alivePeers = newPeers
				Log.Debug().Msgf("Alive peers updated: %v", newPeers)
			}
		}
	}()
}

// mergeRequests merges two Request instances using confirmation logic.
func (handler *RequestHandler) mergeRequests(localReq, remoteReq Request) Request {
	// Adjust state based on priority.
	if localReq.State == REQ_None && remoteReq.State == REQ_Completed {
		// Ignore remote state.
	} else if localReq.State < remoteReq.State {
		localReq.State = remoteReq.State
	} else if localReq.State == REQ_Completed && remoteReq.State == REQ_None {
		localReq.State = remoteReq.State
	}

	// If the request is unconfirmed or completed, try to confirm it.
	if localReq.State == REQ_Completed || localReq.State == REQ_Unconfirmed {
		localReq.ConsensusPeers = MergeStringArrays(localReq.ConsensusPeers, remoteReq.ConsensusPeers)
		localReq = confirmRequest(localReq, handler.localID, handler.alivePeers)
	}

	return localReq
}

// updateLocalRequestMap updates the requestMap for a specific remote node.
func (handler *RequestHandler) updateLocalRequestMap(remoteID string, incomingArr RequestArray) {
	if existingArr, exists := handler.requestMap[remoteID]; exists {
		handler.requestMap[remoteID] = handler.mergeRequestArrays(existingArr, incomingArr)
	} else {
		handler.requestMap[remoteID] = incomingArr
	}
}

// mergeRequestArrays iterates through the RequestArray to merge two arrays.
func (handler *RequestHandler) mergeRequestArrays(local, remote RequestArray) RequestArray {
	merged := local
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			merged[floor][btn] = handler.mergeRequests(local[floor][btn], remote[floor][btn])
		}
	}
	return merged
}

// MergeStringArrays merges two slices of strings ensuring uniqueness.
func MergeStringArrays(arr1 []string, arr2 []string) []string {
	unique := make(map[string]bool)
	var merged []string
	for _, s := range arr1 {
		if !unique[s] {
			unique[s] = true
			merged = append(merged, s)
		}
	}
	for _, s := range arr2 {
		if !unique[s] {
			unique[s] = true
			merged = append(merged, s)
		}
	}
	return merged
}

// containsID checks if a given id exists in the slice.
func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// confirmRequest confirms a request if all nodes have confirmed.
func confirmRequest(req Request, localID string, allNodes []string) Request {
	if !containsID(req.ConsensusPeers, localID) {
		req.ConsensusPeers = append(req.ConsensusPeers, localID)
	}
	for _, node := range allNodes {
		if !containsID(req.ConsensusPeers, node) {
			return req
		}
	}
	switch req.State {
	case REQ_Completed:
		req.State = REQ_None
	case REQ_Unconfirmed:
		req.State = REQ_Confirmed
	}
	req.ConsensusPeers = []string{}
	return req
}
