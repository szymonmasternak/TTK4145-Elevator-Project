package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevevent"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevnet"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
)

// Logger instance.
var Log = logger.GetLogger()

// ---------------------------------------------------------------------
// Hall request assigner types (merged from elevhallrequestassigner package)
// ---------------------------------------------------------------------

// HallRequestAssignerElevatorState is the simplified state used for the hall assignment.
type HallRequestAssignerElevatorState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

// HallRequestAssignerInput is the complete input sent to the external assignment executable.
type HallRequestAssignerInput struct {
	HallRequests [][2]bool                                   `json:"hallRequests"`
	States       map[string]HallRequestAssignerElevatorState `json:"states"`
}

// ---------------------------------------------------------------------
// Merged RequestHandler
// This single handler manages both confirmation logic and hall request assignment.
// ---------------------------------------------------------------------

type RequestHandler struct {
	// Common fields.
	localID string

	// Request confirmation fields.
	requestMap           elevconsts.RequestConfirmationMap
	alivePeers           []string
	requestUpdateChannel <-chan elevconsts.RequestMessage
	inboundArrayChannel  <-chan elevconsts.RequestArrayMessage
	outboundArrayChannel chan<- elevconsts.RequestArrayMessage
	lastSent             elevconsts.RequestConfirmationMap

	// Hall request assigner fields.
	localNetwork *elevnet.ElevatorNetwork
	eventChannel chan<- elevevent.ElevatorEvent

	// For calling the external hall assigner executable.
	executableVersion string

	// Mutex for protecting shared resources.
	mu sync.Mutex
}

// NewRequestHandler creates a new merged RequestHandler.
func NewRequestHandler(
	localID string,
	requestMsgChannel <-chan elevconsts.RequestMessage,
	inboundArrayChannel <-chan elevconsts.RequestArrayMessage,
	outboundArrayChannel chan<- elevconsts.RequestArrayMessage,
	localNetwork *elevnet.ElevatorNetwork,
	eventChannel chan<- elevevent.ElevatorEvent,
) *RequestHandler {
	reqMap := NewRequestConfirmationMap(localID)
	lastSent := make(elevconsts.RequestConfirmationMap)
	lastSent[localID] = reqMap[localID]

	// Broadcast the initial local state.
	outboundArrayChannel <- elevconsts.RequestArrayMessage{
		Identifier:   localID,
		RequestArray: reqMap[localID],
	}

	return &RequestHandler{
		localID:              localID,
		requestMap:           reqMap,
		alivePeers:           []string{localID},
		requestUpdateChannel: requestMsgChannel,
		inboundArrayChannel:  inboundArrayChannel,
		outboundArrayChannel: outboundArrayChannel,
		lastSent:             lastSent,
		localNetwork:         localNetwork,
		eventChannel:         eventChannel,
		executableVersion:    getExecutableVersion(),
	}
}

// Start launches two concurrent loops:
//  1. The confirmation loop listens for local and remote RequestArrayMessages and broadcasts updates.
//  2. The hall assignment loop periodically collects the local elevator state and current confirmed requests,
//     calls the external hall request assigner, and then sends an event with the optimal hall requests.
func (handler *RequestHandler) Start(ctx context.Context, wg *sync.WaitGroup) {
	// Start request confirmation loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		Log.Debug().Msg("Merged RequestHandler: confirmation loop started")
		// Broadcast initial local state.
		//handler.broadcastIfUpdated(handler.localID)

		confirmationTicker := time.NewTicker(1 * time.Second)
		defer confirmationTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msg("Merged RequestHandler: confirmation loop stopping")
				return

			// Process inbound RequestArrayMessages.
			case incomingMsg := <-handler.inboundArrayChannel:
				//Log.Debug().Msgf("Merged RequestHandler: received RequestArrayMessage from %s: %v", incomingMsg.Identifier, incomingMsg.RequestArray)
				// Update alive peers under lock.
				handler.mu.Lock()
				handler.alivePeers = handler.localNetwork.GetAliveNodes()
				//Log.Debug().Msgf("Merged RequestHandler: alive peers: %v", handler.alivePeers)
				handler.mu.Unlock()

				if incomingMsg.Identifier == "" {
					Log.Error().Msg("Empty identifier received in RequestArrayMessage")
					continue
				}
				handler.updateLocalRequestMap(incomingMsg.Identifier, incomingMsg.RequestArray)
				handler.broadcastIfUpdated(incomingMsg.Identifier)

			// Process local RequestMessages.
			case reqMsg := <-handler.requestUpdateChannel:
				handler.mu.Lock()
				handler.alivePeers = handler.localNetwork.GetAliveNodes()
				//Log.Debug().Msgf("RequestHandler: received local RequestMessage: %v", reqMsg)
				tempArr := handler.requestMap[handler.localID]
				// Update the local RequestArray based on the incoming RequestMessage.
				if tempArr[reqMsg.Floor][reqMsg.Button].State <= elevconsts.REQ_None && reqMsg.State == elevconsts.REQ_Unconfirmed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = elevconsts.REQ_Unconfirmed
					tempArr[reqMsg.Floor][reqMsg.Button].ConsensusPeers = []string{handler.localID}
				} else if tempArr[reqMsg.Floor][reqMsg.Button].State == elevconsts.REQ_Confirmed && reqMsg.State == elevconsts.REQ_Completed {
					tempArr[reqMsg.Floor][reqMsg.Button].State = elevconsts.REQ_Completed
					tempArr[reqMsg.Floor][reqMsg.Button].ConsensusPeers = []string{handler.localID}
				}
				handler.requestMap[handler.localID] = tempArr
				//Log.Debug().Msgf("RequestHandler: updated local RequestArray: %v", handler.requestMap[handler.localID])

				// If we are alone, update our own map.
				if len(handler.alivePeers) == 1 {
					// Release the lock before calling updateLocalRequestMap (which locks internally) to avoid deadlock.
					handler.mu.Unlock()
					handler.updateLocalRequestMap(handler.localID, handler.requestMap[handler.localID])
				} else {
					handler.mu.Unlock()
				}
				handler.outboundArrayChannel <- elevconsts.RequestArrayMessage{
					Identifier:   handler.localID,
					RequestArray: handler.requestMap[handler.localID],
				}
				handler.broadcastIfUpdated(handler.localID)
			case <-confirmationTicker.C:
				// Periodically broadcast the local state.
				handler.mu.Lock()
				for peerID, reqs := range handler.requestMap {
					handler.outboundArrayChannel <- elevconsts.RequestArrayMessage{
						Identifier:   peerID,
						RequestArray: reqs,
					}
				}
				handler.mu.Unlock()
			}

		}
	}()

	// Start hall assignment loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		Log.Debug().Msg("Merged RequestHandler: hall assignment loop started")
		assignerTicker := time.NewTicker(1000 * time.Millisecond)
		defer assignerTicker.Stop()

		//var latestState elevstate.ElevatorState
		for {
			select {
			case <-ctx.Done():
				Log.Warn().Msg("Merged RequestHandler: hall assignment loop stopping")
				return

			case <-assignerTicker.C:
				handler.mu.Lock()
				latestRequestMap := handler.requestMap
				handler.mu.Unlock()
				//Log.Debug().Msgf("RequestHandler: latest request map: %v", latestRequestMap)

				// Gather the network-wide elevator states.
				stateMap := handler.localNetwork.GetElevatorStateMap()
				//Log.Debug().Msgf("Merged RequestHandler: elevator state map: %v", stateMap)
				//Log.Debug().Msgf("Merged RequestHandler: elevator state map: %v", stateMap)
				//latestState = stateMap[handler.localID]
				//latestState.ConfirmedRequests = updatedRequests
				for peerID, reqs := range latestRequestMap {
					updatedRequests := [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{}
					for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
						for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
							if reqs[floor][btn].State == elevconsts.REQ_Confirmed {
								updatedRequests[floor][btn] = 1
							}
						}
					}
					tempState := stateMap[peerID]
					tempState.ConfirmedRequests = updatedRequests
					stateMap[peerID] = tempState
				}
				//Log.Debug().Msgf("RequestHandler: updated elevator state map: %v", stateMap)
				// If there is more than one elevator, assign hall requests.
				if len(stateMap) > 1 {
					input := getHallRequestAssignerInput(stateMap)
					//Log.Debug().Msgf("RequestHandler: hall request assigner input: %v", input)
					optimalHallRequests := getOptimalHallRequests(handler.executableVersion, input)
					//Log.Debug().Msgf("RequestHandler: optimal hall requests: %v", optimalHallRequests)
					if optimalHallRequests == nil {
						Log.Warn().Msg("Merged RequestHandler: optimal hall requests came back nil")
						continue
					}
					optimalLocalRequests := getOptimalLocalRequests(optimalHallRequests, stateMap[handler.localID].ConfirmedRequests, handler.localID)
					//Log.Debug().Msgf("Merged RequestHandler: optimal local requests: %v", optimalLocalRequests)
					handler.eventChannel <- elevevent.ElevatorEvent{
						Value: elevevent.UpdateHallRequestsEvent{Requests: optimalLocalRequests},
					}

				} else {
					// If only one elevator exists, send the local requests as is.
					handler.eventChannel <- elevevent.ElevatorEvent{
						Value: elevevent.UpdateHallRequestsEvent{Requests: stateMap[handler.localID].ConfirmedRequests},
					}
				}
			}
		}
	}()
}

// ---------------------------------------------------------------------
// Helper functions for request confirmation
// ---------------------------------------------------------------------

// NewRequestConfirmationMap initializes the RequestConfirmationMap with the local node.
func NewRequestConfirmationMap(localID string) elevconsts.RequestConfirmationMap {
	reqArr := elevconsts.RequestArray{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			reqArr[floor][btn] = elevconsts.Request{
				State:          elevconsts.REQ_Initial,
				ConsensusPeers: []string{},
			}
		}
	}
	requestMap := make(elevconsts.RequestConfirmationMap)
	requestMap[localID] = reqArr
	Log.Debug().Msgf("Initialized RequestConfirmationMap for %s: %v", localID, requestMap)
	return requestMap
}

// broadcastIfUpdated sends an updated RequestArray for the given peer if it differs from the last sent.
func (handler *RequestHandler) broadcastIfUpdated(peerID string) {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	current := handler.requestMap[peerID]
	last, exists := handler.lastSent[peerID]
	if !exists || !requestArrayEqual(current, last) {
		//Log.Debug().Msgf("Merged RequestHandler: broadcasting updated RequestArray for %s: %v", peerID, current)
		handler.outboundArrayChannel <- elevconsts.RequestArrayMessage{
			Identifier:   peerID,
			RequestArray: current,
		}
		// Update lastSent.
		handler.lastSent[peerID] = current
	}
}

func (handler *RequestHandler) updateLocalRequestMap(remoteID string, incomingArr elevconsts.RequestArray) {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if existingArr, exists := handler.requestMap[remoteID]; exists {
		handler.requestMap[remoteID] = handler.mergeRequestArrays(existingArr, incomingArr)
	} else {
		// Initialize a default RequestArray for the remote node.
		defaultArrMap := NewRequestConfirmationMap(remoteID)
		defaultArr := defaultArrMap[remoteID]
		handler.requestMap[remoteID] = handler.mergeRequestArrays(defaultArr, incomingArr)
	}
}

func (handler *RequestHandler) mergeRequestArrays(local, remote elevconsts.RequestArray) elevconsts.RequestArray {
	merged := local
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			merged[floor][btn] = handler.mergeRequests(local[floor][btn], remote[floor][btn])
		}
	}
	return merged
}

// mergeRequests merges two Request instances using confirmation logic.
func (handler *RequestHandler) mergeRequests(localReq, remoteReq elevconsts.Request) elevconsts.Request {
	// Adjust state based on priority.
	if localReq.State == elevconsts.REQ_None && remoteReq.State == elevconsts.REQ_Completed {
		// Ignore remote state.
	} else if localReq.State < remoteReq.State {
		localReq.State = remoteReq.State
	} else if localReq.State == elevconsts.REQ_Completed && remoteReq.State == elevconsts.REQ_None {
		localReq.State = remoteReq.State
	}

	// If the request is unconfirmed or completed, merge consensus peers and try to confirm.
	if localReq.State == elevconsts.REQ_Completed || localReq.State == elevconsts.REQ_Unconfirmed {
		localReq.ConsensusPeers = MergeStringArrays(localReq.ConsensusPeers, remoteReq.ConsensusPeers)
		localReq = confirmRequest(localReq, handler.localID, handler.alivePeers)
	}

	return localReq
}

// requestArrayEqual compares two RequestArrays.
func requestArrayEqual(a, b elevconsts.RequestArray) bool {
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
func confirmRequest(req elevconsts.Request, localID string, allNodes []string) elevconsts.Request {
	if !containsID(req.ConsensusPeers, localID) {
		req.ConsensusPeers = append(req.ConsensusPeers, localID)
	}
	for _, node := range allNodes {
		if !containsID(req.ConsensusPeers, node) {
			return req
		}
	}
	switch req.State {
	case elevconsts.REQ_Completed:
		req.State = elevconsts.REQ_None
	case elevconsts.REQ_Unconfirmed:
		req.State = elevconsts.REQ_Confirmed
	}
	req.ConsensusPeers = []string{}
	return req
}

// ---------------------------------------------------------------------
// Helper functions for hall request assignment
// ---------------------------------------------------------------------

// getExecutableVersion returns the platform-specific executable name.
func getExecutableVersion() string {
	var executableName string
	switch runtime.GOOS {
	case "linux":
		executableName = "hall_request_assigner"
	case "windows":
		executableName = "hall_request_assigner.exe"
	case "darwin":
		executableName = "macOS_hall_request_assigner"
	default:
		panic("OS not supported")
	}
	return executableName
}

// getHallRequestAssignerElevatorState converts an elevator state into the assigner’s state.
func getHallRequestAssignerElevatorState(elevatorState *elevstate.ElevatorState) HallRequestAssignerElevatorState {
	var cabRequests [elevconsts.N_FLOORS]bool
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		if elevatorState.ConfirmedRequests[floor][elevconsts.Cab] != 0 {
			cabRequests[floor] = true
		}
	}
	return HallRequestAssignerElevatorState{
		Behavior:    elevatorState.Behaviour.String(),
		Floor:       elevatorState.Floor,
		Direction:   elevatorState.Dirn.String(),
		CabRequests: cabRequests[:],
	}
}

func getHallRequestAssignerInput(elevatorStateMap map[string]elevstate.ElevatorState) HallRequestAssignerInput {
	hallRequests := [elevconsts.N_FLOORS][2]bool{}
	// Loop through all elevators in the state map.
	for _, elevatorState := range elevatorStateMap {
		for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
			hallRequests[floor][elevconsts.HallUp] = hallRequests[floor][elevconsts.HallUp] ||
				(elevatorState.ConfirmedRequests[floor][elevconsts.HallUp] != 0)
			hallRequests[floor][elevconsts.HallDown] = hallRequests[floor][elevconsts.HallDown] ||
				(elevatorState.ConfirmedRequests[floor][elevconsts.HallDown] != 0)
		}
	}

	states := make(map[string]HallRequestAssignerElevatorState)
	for id, state := range elevatorStateMap {
		states[id] = getHallRequestAssignerElevatorState(&state)
	}
	return HallRequestAssignerInput{
		HallRequests: hallRequests[:],
		States:       states,
	}
}

// getOptimalHallRequests executes the external hall request assigner.
func getOptimalHallRequests(executableVersion string, input HallRequestAssignerInput) map[string][][2]bool {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error:", err)
		return nil
	}

	ret, err := exec.Command("../libs/hall_request_assigner/"+executableVersion, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error:", err)
		fmt.Println(string(ret))
		return nil
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error:", err)
		return nil
	}
	return *output
}

// getOptimalLocalRequests returns the optimal local request array for the given localID.
func getOptimalLocalRequests(optimalHallRequests map[string][][2]bool, originalLocalRequests [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int, localID string) [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int {
	localRequests := [elevconsts.N_FLOORS][elevconsts.N_BUTTONS]int{}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		localRequests[floor][elevconsts.HallUp] = toInt(optimalHallRequests[localID][floor][elevconsts.HallUp])
		localRequests[floor][elevconsts.HallDown] = toInt(optimalHallRequests[localID][floor][elevconsts.HallDown])
		localRequests[floor][elevconsts.Cab] = originalLocalRequests[floor][elevconsts.Cab]
	}
	return localRequests
}

// toInt converts a boolean to int.
func toInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
