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

	state := elevstate.ElevatorState{}

	msg := MakeElevatorMessage(&metaData, &state)

	stateInChannel := make(chan elevstate.ElevatorState)
	stateOutChannel := make(chan elevstate.ElevatorState)

	broadcastingPeriod := 10 * time.Millisecond
	listeningTimeout := broadcastingPeriod * 2

	network := NewElevatorNetwork(&metaData, &state, stateInChannel, stateOutChannel)
	network.Broadcast.Start(broadcastingPeriod)
	defer network.Broadcast.Stop()

	network.Listen.Start()
	defer network.Listen.Stop()

	timerticker := time.NewTimer(listeningTimeout)
	defer timerticker.Stop()

	select {
	case nodeFound := <-network.Listen.ElevatorsFoundOnNetwork:
		if nodeFound != msg {
			t.Errorf("Wrong Elevator found on network = %s, expected %s", nodeFound.ElevatorData.String(), metaData.String())
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
	time.Sleep(300 * time.Millisecond) //used to make the node't timestamp bigger, simulating disconnection

	deadline := time.Now().Add(500 * time.Millisecond)
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
