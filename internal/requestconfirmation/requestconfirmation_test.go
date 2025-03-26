package requestconfirmation

import (
	"reflect"
	"testing"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
)

// TestMergeStringArrays tests that merging two string slices produces unique results.
func TestMergeStringArrays(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"b", "c"}
	result := MergeStringArrays(a, b)
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	// Test with an empty slice.
	result = MergeStringArrays([]string{}, []string{"x"})
	expected = []string{"x"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestContainsID verifies that containsID correctly detects the presence or absence of an ID.
func TestContainsID(t *testing.T) {
	arr := []string{"node1", "node2"}
	if !containsID(arr, "node1") {
		t.Error("expected node1 to be present")
	}
	if containsID(arr, "node3") {
		t.Error("expected node3 not to be present")
	}
}

// TestConfirmRequest tests the confirmRequest function for the proper state transitions.
func TestConfirmRequest(t *testing.T) {
	allNodes := []string{"node1", "node2"}

	// Case 1: For REQ_Unconfirmed, if the local node is missing but adding it completes the set,
	// state becomes REQ_Confirmed.
	req := Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	confirmed := confirmRequest(req, "node1", allNodes)
	if confirmed.State != REQ_Confirmed {
		t.Errorf("expected state REQ_Confirmed, got %v", confirmed.State)
	}
	if len(confirmed.ConsensusPeers) != 0 {
		t.Errorf("expected consensus peers to be cleared, got %v", confirmed.ConsensusPeers)
	}

	// Case 2: Not all nodes confirmed (missing one), so state should remain unchanged
	// (apart from adding localID).
	allNodes2 := []string{"node1", "node2", "node3"}
	req2 := Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	confirmed2 := confirmRequest(req2, "node1", allNodes2)
	if confirmed2.State != REQ_Unconfirmed {
		t.Errorf("expected state REQ_Unconfirmed, got %v", confirmed2.State)
	}
	if !containsID(confirmed2.ConsensusPeers, "node1") || !containsID(confirmed2.ConsensusPeers, "node2") {
		t.Errorf("expected consensus peers to include node1 and node2, got %v", confirmed2.ConsensusPeers)
	}

	// Case 3: For REQ_Completed, when all nodes confirm, the state should transition to REQ_None.
	req3 := Request{
		State:          REQ_Completed,
		ConsensusPeers: []string{"node2"},
	}
	confirmed3 := confirmRequest(req3, "node1", allNodes)
	if confirmed3.State != REQ_None {
		t.Errorf("expected state REQ_None, got %v", confirmed3.State)
	}
	if len(confirmed3.ConsensusPeers) != 0 {
		t.Errorf("expected consensus peers to be cleared, got %v", confirmed3.ConsensusPeers)
	}
}

// TestMergeRequests verifies that merging two Request instances using the RequestHandler method
// produces the expected state and consensus.
func TestMergeRequests(t *testing.T) {
	// Create a dummy RequestHandler.
	handler := &RequestHandler{
		localID:    "node1",
		alivePeers: []string{"node1", "node2", "node3"},
	}
	localReq := Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	remoteReq := Request{
		State:          REQ_Completed,
		ConsensusPeers: []string{"node3"},
	}
	merged := handler.mergeRequests(localReq, remoteReq)
	if merged.State != REQ_None {
		t.Errorf("expected state REQ_None after full confirmation, got %v", merged.State)
	}
	if len(merged.ConsensusPeers) != 0 {
		t.Errorf("expected consensus peers to be cleared, got %v", merged.ConsensusPeers)
	}

	// Test merging when local state is REQ_None and remote is REQ_Completed.
	localReq2 := Request{
		State:          REQ_None,
		ConsensusPeers: []string{},
	}
	remoteReq2 := Request{
		State:          REQ_Completed,
		ConsensusPeers: []string{"node2"},
	}
	handler.alivePeers = []string{"node1", "node2"}
	merged2 := handler.mergeRequests(localReq2, remoteReq2)
	if merged2.State != REQ_None {
		t.Errorf("expected state REQ_None, got %v", merged2.State)
	}
}

// TestMergeRequestArrays tests that two RequestArrays merge correctly using the handler's method.
func TestMergeRequestArrays(t *testing.T) {
	handler := &RequestHandler{
		localID:    "node1",
		alivePeers: []string{"node1", "node2", "node3"},
	}
	var arr1, arr2 RequestArray
	// Set a request in arr1 (e.g., at floor 1, button 1).
	arr1[1][1] = Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	// Set a conflicting request in arr2 at the same position.
	arr2[1][1] = Request{
		State:          REQ_Completed,
		ConsensusPeers: []string{"node3"},
	}
	merged := handler.mergeRequestArrays(arr1, arr2)
	if merged[1][1].State != REQ_None {
		t.Errorf("expected merged state REQ_None, got %v", merged[1][1].State)
	}
	if len(merged[1][1].ConsensusPeers) != 0 {
		t.Errorf("expected consensus peers to be cleared, got %v", merged[1][1].ConsensusPeers)
	}
}

// TestUpdateLocalRequestMap verifies that updateLocalRequestMap correctly updates the requestMap.
func TestUpdateLocalRequestMap(t *testing.T) {
	localID := "node1"
	handler := &RequestHandler{
		localID:    localID,
		requestMap: NewRequestConfirmationMap(localID),
		alivePeers: []string{"node1"},
	}
	var incomingArr RequestArray
	incomingArr[0][0] = Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	handler.updateLocalRequestMap("node2", incomingArr)
	if _, exists := handler.requestMap["node2"]; !exists {
		t.Error("expected remote node entry to be added")
	}

	// Update with a different state and consensus to test merging.
	incomingArr[0][0] = Request{
		State:          REQ_Completed,
		ConsensusPeers: []string{"node2", "node1"},
	}
	handler.updateLocalRequestMap("node2", incomingArr)
	result := handler.requestMap["node2"][0][0]
	if result.State != REQ_None {
		t.Errorf("expected merged state REQ_None, got %v", result.State)
	}
	if len(result.ConsensusPeers) != 0 {
		t.Errorf("expected consensus peers to be cleared, got %v", result.ConsensusPeers)
	}
}

// TestNewRequestConfirmationMap checks that the local RequestConfirmationMap is initialized correctly.
func TestNewRequestConfirmationMap(t *testing.T) {
	localID := "node1"
	reqMap := NewRequestConfirmationMap(localID)
	if _, exists := reqMap[localID]; !exists {
		t.Errorf("expected localID %s in map", localID)
	}
	for floor := 0; floor < elevconsts.N_FLOORS; floor++ {
		for btn := 0; btn < elevconsts.N_BUTTONS; btn++ {
			if reqMap[localID][floor][btn].State != REQ_Initial {
				t.Errorf("expected initial state at floor %d, button %d; got %v", floor, btn, reqMap[localID][floor][btn].State)
			}
		}
	}
}

// TestStart_Basic simulates one cycle of the RequestHandler processing an inbound RequestArrayMessage.
// Since Start spawns a goroutine, we use buffered channels and a timeout to test one cycle.
func TestStart_Basic(t *testing.T) {
	localID := "node1"
	reqMsgChannel := make(chan RequestMessage)
	inboundChannel := make(chan RequestArrayMessage, 2)
	outboundChannel := make(chan RequestArrayMessage, 2)
	alivePeersChannel := make(chan []string)

	handler := NewRequestHandler(localID, reqMsgChannel, inboundChannel, outboundChannel, alivePeersChannel)
	// Ensure alivePeers is set.
	handler.alivePeers = []string{localID}

	// Start the handler (spawns a goroutine).
	handler.Start()

	// Create a dummy inbound message from a remote node.
	var remoteArr RequestArray
	remoteArr[0][0] = Request{
		State:          REQ_Unconfirmed,
		ConsensusPeers: []string{"node2"},
	}
	inboundChannel <- RequestArrayMessage{Identifier: "node2", RequestArray: remoteArr}

	// Wait for an outbound message from the handler.
	select {
	case msg := <-outboundChannel:
		if msg.Identifier != localID {
			t.Errorf("expected identifier %s, got %s", localID, msg.Identifier)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for outbound RequestArrayMessage")
	}
}
