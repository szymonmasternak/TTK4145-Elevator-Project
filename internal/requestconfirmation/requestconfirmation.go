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
	State          RequestState
	ConsensusPeers []string
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
	Identifier   string
	RequestArray RequestArray
}

// RequestHandler manages local state and communications with other nodes.
type RequestHandler struct {
	localID string
	// requestMap holds the local state and remote nodes' states.
	requestMap RequestConfirmationMap
	alivePeers []string

	requestUpdateChannel <-chan RequestMessage      // Local messages (e.g., button presses)
	inboundArrayChannel  <-chan RequestArrayMessage // Inbound messages from remote nodes
	outboundArrayChannel chan<- RequestArrayMessage // Outbound messages for broadcasting local state
	alivePeersChannel    <-chan []string
}

// NewRequestHandler creates a new RequestHandler instance.
func NewRequestHandler(
	localID string,
	requestMsgChannel <-chan RequestMessage,
	inboundArrayChannel <-chan RequestArrayMessage,
	outboundArrayChannel chan<- RequestArrayMessage,
	alivePeersChannel <-chan []string,
) *RequestHandler {
	return &RequestHandler{
		localID:              localID,
		requestMap:           NewRequestConfirmationMap(localID),
		alivePeers:           []string{localID},
		requestUpdateChannel: requestMsgChannel,
		inboundArrayChannel:  inboundArrayChannel,
		outboundArrayChannel: outboundArrayChannel,
		alivePeersChannel:    alivePeersChannel,
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
	return requestMap
}

// Start listens for inbound RequestArrayMessages and local RequestMessages,
// updates the internal state accordingly, and broadcasts changes via the outbound channel.
func (handler *RequestHandler) Start() {
	Log.Debug().Msgf("RequestConfirmer started")
	confirmationTicker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			// Process inbound messages from remote nodes.
			case incomingMsg := <-handler.inboundArrayChannel:
				Log.Debug().Msgf("Inbound RequestArrayMessage received from %s", incomingMsg.Identifier)
				handler.alivePeers = MergeStringArrays(handler.alivePeers, []string{incomingMsg.Identifier})
				handler.updateLocalRequestMap(incomingMsg.Identifier, incomingMsg.RequestArray)
				Log.Debug().Msgf("Local map for %s: %v", handler.localID, handler.requestMap[handler.localID])
				// Broadcast updated local state.
				handler.outboundArrayChannel <- RequestArrayMessage{
					Identifier:   handler.localID,
					RequestArray: handler.requestMap[handler.localID],
				}

			// Process local request messages.
			case reqMsg := <-handler.requestUpdateChannel:
				Log.Debug().Msgf("Local RequestMessage received")
				tempArr := handler.requestMap[handler.localID]
				if tempArr[reqMsg.Floor][reqMsg.Button].State <= REQ_None && reqMsg.State == REQ_Unconfirmed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Unconfirmed
				} else if tempArr[reqMsg.Floor][reqMsg.Button].State == REQ_Confirmed && reqMsg.State == REQ_Completed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Completed
				}
				handler.requestMap[handler.localID] = tempArr
				handler.outboundArrayChannel <- RequestArrayMessage{
					Identifier:   handler.localID,
					RequestArray: handler.requestMap[handler.localID],
				}
				Log.Debug().Msgf("Updated local map: %v", handler.requestMap[handler.localID])

			// Periodic update using the ticker.
			case <-confirmationTicker.C:
				// For example, only update if only the local node is active.
				if len(handler.alivePeers) == 1 {
					handler.updateLocalRequestMap(handler.localID, handler.requestMap[handler.localID])
					handler.outboundArrayChannel <- RequestArrayMessage{
						Identifier:   handler.localID,
						RequestArray: handler.requestMap[handler.localID],
					}
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
	// Merge the consensus lists.
	localReq.ConsensusPeers = MergeStringArrays(localReq.ConsensusPeers, remoteReq.ConsensusPeers)

	// Adjust state based on priority.
	if localReq.State == REQ_None && remoteReq.State == REQ_Completed {
		// Ignore remote state.
	} else if localReq.State < remoteReq.State {
		localReq.State = remoteReq.State
	} else if localReq.State == REQ_Completed && remoteReq.State == REQ_None {
		localReq.State = remoteReq.State
	}

	// If request is unconfirmed or completed, try to confirm it.
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
