package elevnet

import (
	"testing"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

func TestStartBroadcastingListening(t *testing.T) {
	metaData := elevmetadata.ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}

	broadcastingPeriod := 10 * time.Millisecond
	listeningTimeout := broadcastingPeriod * 5 // ✅ Increase timeout to allow broadcast to be received

	stateInChannel := make(chan elevstate.ElevatorState, 10)
	stateOutChannel := make(chan elevstate.ElevatorState, 10)

	network := NewElevatorNetwork(&metaData, stateOutChannel, stateInChannel)
	network.Broadcast.Start(broadcastingPeriod)
	defer network.Broadcast.Stop()

	network.Listen.Start()
	defer network.Listen.Stop()

	timerticker := time.NewTimer(listeningTimeout)
	defer timerticker.Stop()

	select {
	case nodeFound := <-network.Listen.ElevatorsFoundOnNetwork:
		if nodeFound != metaData {
			t.Errorf("Wrong Elevator found on network = %s, expected %s", nodeFound.String(), metaData.String())
		}
		timerticker.Stop()
		return
	case <-timerticker.C:
		t.Errorf("Timed out waiting for elevator to be found on network")
		return
	}
}

func elevatorArrayContains(nl *ElevNetListen, identifier string) bool {
	for _, i := range nl.elevatorArray {
		if i.ElevatorData.Identifier == identifier {
			return true
		}
	}
	return false
}

func getTimestampFor(nl *ElevNetListen, identifier string) time.Time {
	for _, i := range nl.elevatorArray {
		if i.ElevatorData.Identifier == identifier {
			return i.timeSeen
		}
	}
	return time.Time{}
}

func countOccurrences(nl *ElevNetListen, identifier string) int {
	count := 0
	for _, i := range nl.elevatorArray {
		if i.ElevatorData.Identifier == identifier {
			count++
		}
	}
	return count
}

func TestAddNodeToList(t *testing.T) {
	// Create the global stateInChannel
	stateInChannel := make(chan elevstate.ElevatorState, 10)

	// Initialize ElevNetListen with all required fields:
	nl := &ElevNetListen{
		ElevatorsFoundOnNetwork: make(chan elevmetadata.ElevMetaData),
		listening:               false,
		startStopCh:             make(chan int),
		conn:                    nil,
		elevMetaData: &elevmetadata.ElevMetaData{
			SoftwareVersion: "dummy",
			IpAddress:       "127.0.0.1",
			PortNumber:      9999,
			Identifier:      "testElevator",
		},
		elevatorArray:  []ElevatorListObject{},
		stateInChannel: stateInChannel,
	}

	// 1st test: add one node to the list
	nodeA := elevmetadata.ElevMetaData{"sdijfoisj", "0.0.0.0", 9999, "elevator1"}
	nl.AddNodeToList(nodeA)
	if !elevatorArrayContains(nl, "elevator1") {
		t.Error("Expected elevator1 to be in the list")
	}

	// 2nd test: update timestamp correctly
	oldTimestamp := getTimestampFor(nl, "elevator1")
	time.Sleep(50 * time.Millisecond)
	nl.AddNodeToList(nodeA)
	newTimestamp := getTimestampFor(nl, "elevator1")
	if !newTimestamp.After(oldTimestamp) {
		t.Error("Expected elevator1's timestamp to be updated")
	}
	if countOccurrences(nl, "elevator1") != 1 {
		t.Error("Expected only one occurrence of elevator1 in the list")
	}

	// 3rd test: add second node
	nodeB := elevmetadata.ElevMetaData{"sodicmxzxj", "0.0.0.0", 9999, "elevator2"}
	nl.AddNodeToList(nodeB)
	if !elevatorArrayContains(nl, "elevator1") || !elevatorArrayContains(nl, "elevator2") {
		t.Error("Expected both elevator1 and elevator2 to be in the list")
	}

	// 4th test: removing node
	time.Sleep(300 * time.Millisecond) // simulate disconnection

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		nl.AddNodeToList(nodeB)
		if !elevatorArrayContains(nl, "elevator1") {
			break // elevator1 removed
		}
		if time.Now().After(deadline) {
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
