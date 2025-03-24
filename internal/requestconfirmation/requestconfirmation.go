package requestconfirmation

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	//"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	//"time"
)

var Log = logger.GetLogger()

type RequestState int

const (
	REQ_Initial     RequestState = -1
	REQ_None                     = 0
	REQ_Unconfirmed              = 1
	REQ_Confirmed                = 2
	REQ_Completed                = 3
)

type Request struct {
	State RequestState
	ConsensusPeers []string
}

type RequestArray [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]Request

type RequestConfirmationMap map[string]RequestArray

type RequestMessage struct {
	Floor int
	Button elevconsts.Button
	State RequestState
}

func NewRequestConfirmationMap(localID string) RequestConfirmationMap {
    reqArr := RequestArray{}
    for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
        for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
            reqArr[floor][btn] = Request{
                State: REQ_Initial,
                ConsensusPeers: []string{},
            }
        }
    }
    
    requestMap := make(RequestConfirmationMap)
    requestMap[localID] = reqArr
    return requestMap
}

func  RequestConfirmer(localID string, requestMsgChannel <-chan RequestMessage, localRequestArray *RequestArray, ElevatorMessageChannel <-chan elevnet.ElevatorMessage) {
	RequestConfirmationMap := NewRequestConfirmationMap(localID)
	alivePeers := []string{}
	//confirmationTicker := time.NewTicker(100 * time.Millisecond)
	//localButtonPressChannel := make(chan elevevent.ButtonPressEvent)
	//newRequestStateChannel := make(chan RequestArray)
	//requestMsgChannel := make(chan RequestMessage)
	for {
		select {
		case incomingMsg := <-ElevatorMessageChannel:
			// incomingMsg.remoteID contains the remote node's ID
			// incomingMsg.arr contains the remote node's RequestArray.
			RequestConfirmationMap = updateLocalRequestConfirmationMapFromIncomingArray(
				RequestConfirmationMap,
				incomingMsg.ElevatorData.Identifier,
				incomingMsg.RequestArray,
				localID,
				alivePeers,
			)
			localRequestArray = (RequestConfirmationMap[localID])
		case msg := <-alivePeersChannel:
			// Update the list of alive peers.
			alivePeers = msg.Peers
		case reqMsg := <-requestMsgChannel:
			tempArr := RequestConfirmationMap[localID]
			if tempArr[reqMsg.Floor][reqMsg.Button].State == REQ_None && reqMsg.State == REQ_Unconfirmed {
				tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Unconfirmed
				RequestConfirmationMap[localID] = tempArr
			} else if tempArr[reqMsg.Floor][reqMsg.Button].State == REQ_Confirmed && reqMsg.State == REQ_Completed {
				tempArr[reqMsg.Floor][reqMsg.Button].State = REQ_Completed
				RequestConfirmationMap[localID] = tempArr
			}

		// case newReq := <-localButtonPressChannel:
		// 	if RequestConfirmationMap[localID][newReq.Floor][newReq.Button].State == REQ_None {
		// 		tempArr := RequestConfirmationMap[localID]
		// 		tempArr[newReq.Floor][newReq.Button].State = REQ_Unconfirmed
		// 		RequestConfirmationMap[localID] = tempArr
		// 	}
		// case reqCompleted := <-localRequestCompletedChannel:
		// 	if RequestConfirmationMap[localID][reqCompleted.Floor][reqCompleted.Button].State == REQ_Confirmed {
		// 		RequestConfirmationMap[localID][reqCompleted.Floor][reqCompleted.Button].State = REQ_Completed
		// 	}else {
		// 		Log.Error().Msgf("Request completed without being confirmed")
		// 	}
		}
	}	


}
// mergeRequests merges two Request instances (local and remote) using the confirmation logic.
func mergeRequests(localReq, remoteReq Request, localID string, allNodes []string) Request {
	// Merge the consensus lists.
	localReq.ConsensusPeers = MergeStringArrays(localReq.ConsensusPeers, remoteReq.ConsensusPeers)

	// Cyclic counter
	if localReq.State == REQ_None && remoteReq.State == REQ_Completed {
		//continue
	} else if localReq.State < remoteReq.State {
		localReq.State = remoteReq.State
	} else if localReq.State == REQ_Completed && remoteReq.State == REQ_None {
		localReq.State = remoteReq.State
	}

	if localReq.State == REQ_Completed || localReq.State == REQ_Unconfirmed {
		localReq.ConsensusPeers = MergeStringArrays(localReq.ConsensusPeers, remoteReq.ConsensusPeers)
		localReq = confirmRequest(localReq, localID, allNodes)
	}
	
	return localReq
}

// confirmRequest merges confirmations and updates the request state
// only when all nodes (as provided by allNodes slice) have confirmed.
func confirmRequest(req Request, localID string, allNodes []string) Request {
    // Ensure the local node has confirmed.
    if !containsID(req.ConsensusPeers, localID) {
        req.ConsensusPeers = append(req.ConsensusPeers, localID)
    }

    // If the number of unique confirmations equals the number of active nodes,
    // then all nodes have confirmed.
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

// updateLocalRequestConfirmationMapFromIncomingArray updates the local RequestConfirmationMap
// for a specific remote node using the incoming RequestArray.
func updateLocalRequestConfirmationMapFromIncomingArray(
	localMap RequestConfirmationMap,
	remoteID string,
	incomingArr RequestArray,
	localID string,
	allNodes []string,
) RequestConfirmationMap {
	// If an entry for remoteID exists, merge the incoming array with the existing one.
	if existingArr, exists := localMap[remoteID]; exists {
		localMap[remoteID] = mergeRequestArrays(existingArr, incomingArr, localID, allNodes)
	} else {
		// Otherwise, simply add the incoming array.
		localMap[remoteID] = incomingArr
	}
	return localMap
}

// // updateLocalRequestConfirmationMap takes the local RequestConfirmationMap
// // and merges in the incoming RequestConfirmationMap using the mergeRequestArrays logic.
// func updateLocalRequestConfirmationMap(
// 	localMap RequestConfirmationMap,
// 	incomingMap RequestConfirmationMap,
// 	localID string,
// 	allNodes []string,
// ) RequestConfirmationMap {
// 	// Iterate over each node in the incoming map.
// 	for nodeID, incomingReqArray := range incomingMap {
// 		if localReqArray, exists := localMap[nodeID]; exists {
// 			// If we already have an entry for this node, merge the two RequestArrays.
// 			localMap[nodeID] = mergeRequestArrays(localReqArray, incomingReqArray, localID, allNodes)
// 		} else {
// 			// If this is a new node, simply add its RequestArray.
// 			localMap[nodeID] = incomingReqArray
// 		}
// 	}
// 	return localMap
// }



// MergeStringArrays merges two slices of strings ensuring uniqueness.
func MergeStringArrays(arr1 []string, arr2 []string) []string {
    unique := make(map[string]bool)
    merged := []string{}
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


// mergeRequestArrays iterates through the entire RequestArray to merge two arrays.
func mergeRequestArrays(local, remote RequestArray, localID string, allNodes []string) RequestArray {
	merged := local
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			merged[floor][btn] = mergeRequests(local[floor][btn], remote[floor][btn], localID, allNodes)
		}
	}
	return merged
}

// containsID checks if an id is already in the slice.
func containsID(ids []string, id string) bool {
    for _, v := range ids {
        if v == id {
            return true
        }
    }
    return false
}

// --- ElevatorState Update Integration ---

// // UpdateState merges a new incoming state (newState) into the local state.
// // It updates the requests array only when every active node (allNodes) has confirmed a request.
// func (es *ElevatorState) UpdateState(newState ElevatorState, allNodes []string, localID string) {
// 	// Merge the request arrays using our helper function.
// 	es.Requests = mergeRequestArrays(es.Requests, newState.Requests, localID, allNodes)

// 	// Update other state fields as necessary.
// 	es.Floor = newState.Floor
// 	es.Dirn = newState.Dirn
// 	es.Behaviour = newState.Behaviour

// 	// Update the button lights based on the updated requests.
// 	es.setAllLightsSequence()
// }
