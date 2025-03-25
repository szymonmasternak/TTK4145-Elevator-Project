package elevnet

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/requestconfirmation"
)

func compareElevatorMessages(a, b ElevatorMessage) bool {
	return reflect.DeepEqual(a.ElevatorData, b.ElevatorData) &&
		reflect.DeepEqual(a.ElevatorState, b.ElevatorState)
}

func TestStartBroadcastingListening(t *testing.T) {

	metaData := elevmetadata.ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}

	state := elevstate.ElevatorState{}

	expectedMsg := MakeElevatorMessage(&metaData, &state)

	stateOutChannel := make(chan elevstate.ElevatorState)
	inboundReqArrayChannel := make(chan requestconfirmation.RequestArrayMessage)
	outboundReqArrayChannel := make(chan requestconfirmation.RequestArrayMessage)

	broadcastingPeriod := 10 * time.Millisecond
	listeningTimeout := broadcastingPeriod * 2

	network := NewElevatorNetwork(&metaData, &state, stateOutChannel, outboundReqArrayChannel, inboundReqArrayChannel)
	network.Broadcast.Start(broadcastingPeriod)
	defer network.Broadcast.Stop()

	network.Listen.Start()
	defer network.Listen.Stop()

	timer := time.NewTimer(listeningTimeout)
	defer timer.Stop()

	select {
	case nodeFound := <-network.Listen.ElevatorsFoundOnNetwork:
		// Use the helper function to compare only the ElevatorData and ElevatorState.
		if !compareElevatorMessages(nodeFound, expectedMsg) {
			t.Errorf("Wrong Elevator found on network = %+v, expected %+v",
				nodeFound.ElevatorData, expectedMsg.ElevatorData)
		}
	case <-timer.C:
		t.Errorf("Timed out waiting for elevator to be found on network")
	}
}

func elevatorArrayContains(nl *ElevNetListen, identifier string) bool {
	for _, i := range nl.elevatorArray {
		if i.msg.ElevatorData.Identifier == identifier {
			return true
		}
	}
	return false
}

func getTimestampFor(nl *ElevNetListen, identifier string) time.Time {
	for _, i := range nl.elevatorArray {
		if i.msg.ElevatorData.Identifier == identifier {
			return i.timeSeen
		}
	}
	return time.Time{}
}

func countOccurrences(nl *ElevNetListen, identifier string) int {
	count := 0
	for _, i := range nl.elevatorArray {
		if i.msg.ElevatorData.Identifier == identifier {
			count++
		}
	}
	return count
}

func TestAddNodeToList(t *testing.T) {
	nl := &ElevNetListen{
		elevatorArray: []ElevatorListObject{},
	}

	// 1st test: add one node to the list
	node1data := elevmetadata.ElevMetaData{"sdijfoisj", "0.0.0.0", 9999, "elevator1"}
	node1state := elevstate.ElevatorState{}
	msg1 := MakeElevatorMessage(&node1data, &node1state)
	nl.AddNodeToList(msg1)
	if !elevatorArrayContains(nl, "elevator1") {
		t.Error("Expected elevator1 to be in the list")
	}

	// 2nd test: update timestamp correctly
	oldTimestamp := getTimestampFor(nl, "elevator1")
	time.Sleep(50 * time.Millisecond)
	nl.AddNodeToList(msg1)
	newTimestamp := getTimestampFor(nl, "elevator1")
	if !newTimestamp.After(oldTimestamp) {
		t.Error("Expected elevator1's timestamp to be updated")
	}
	if countOccurrences(nl, "elevator1") != 1 {
		t.Error("Expected only one occurrence of elevator1 in the list")
	}

	// 3rd test: add second node
	node2data := elevmetadata.ElevMetaData{"sodicmxzxj", "0.0.0.0", 9999, "elevator2"}
	node2state := elevstate.ElevatorState{}
	msg2 := MakeElevatorMessage(&node2data, &node2state)
	nl.AddNodeToList(msg2)
	if !elevatorArrayContains(nl, "elevator1") || !elevatorArrayContains(nl, "elevator2") {
		t.Error("Expected both elevator1 and elevator2 to be in the list")
	}

	// 4th test: removing node
	time.Sleep(1000 * time.Millisecond) //used to make the node't timestamp bigger, simulating disconnection

	deadline := time.Now().Add(2000 * time.Millisecond)
	for {
		nl.AddNodeToList(msg2)
		if !elevatorArrayContains(nl, "elevator1") {
			// if elevator1 is removed, exit the loop.
			break
		}
		if time.Now().After(deadline) {
			// if elevator1 is still there after 500ms give up
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if elevatorArrayContains(nl, "elevator1") {
		t.Error("Expected elevator1 to be removed as stale")
	}
	if !elevatorArrayContains(nl, "elevator2") {
		t.Error("Expected elevator2 to remain in the list")
	}

}

func TestAckResponse(t *testing.T) {
	// Use IPv4 binding.
	meta := elevmetadata.ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "127.0.0.1", // Force IPv4 binding.
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}
	var dummyState elevstate.ElevatorState

	outChan := make(chan elevstate.ElevatorState)
	requestChan := make(chan requestconfirmation.RequestArrayMessage)

	listener := NewElevNetListen(&meta, &dummyState, outChan, requestChan)
	// Buffer the broadcast channel to avoid blocking.
	listener.ElevatorsFoundOnNetwork = make(chan ElevatorMessage, 10)

	err := listener.Start()
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	// Allow some time for the listener goroutines to initialize.
	time.Sleep(100 * time.Millisecond)

	localAddrStr := listener.conn.LocalAddr().String()
	t.Logf("Listener bound to: %s", localAddrStr)

	// Force IPv4 for the client.
	clientAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve client address: %v", err)
	}
	serverAddr, err := net.ResolveUDPAddr("udp4", localAddrStr)
	if err != nil {
		t.Fatalf("Failed to resolve server address: %v", err)
	}
	clientConn, err := net.DialUDP("udp4", clientAddr, serverAddr)
	if err != nil {
		t.Fatalf("Failed to dial UDP: %v", err)
	}
	defer clientConn.Close()

	// Create a test ElevatorMessage with AckMsg.Acknowledged false and a known ID (42).
	testMsg := ElevatorMessage{
		ElevatorData:  elevmetadata.ElevMetaData{}, // Dummy data.
		ElevatorState: dummyState,
	}

	jsonData, err := json.Marshal(testMsg)
	if err != nil {
		t.Fatalf("Failed to marshal test message: %v", err)
	}

	_, err = clientConn.Write(jsonData)
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	if err := listener.Stop(); err != nil {
		t.Errorf("Error stopping listener: %v", err)
	}
}
