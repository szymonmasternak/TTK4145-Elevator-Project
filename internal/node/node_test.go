package node

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	node1 := Node{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		NodeNumber:      1,
		DeviceType:      "unknown",
	}

	jsonString := "{\"software_version\":\"smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3\",\"ip_address\":\"0.0.0.0\",\"port_number\":9999,\"node_number\":1,\"device_type\":\"unknown\"}"

	if node1.String() != jsonString {
		t.Errorf("String() = %s, expected %s", node1.String(), jsonString)
	}
}

func TestGetIPAddressPort(t *testing.T) {
	node1 := Node{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		NodeNumber:      1,
		DeviceType:      "unknown",
	}

	if node1.GetIPAddressPort() != "0.0.0.0:9999" {
		t.Errorf("String() = %s, expected 0.0.0.0:9999", node1.String())
	}
}

func TestStartBroadcastingListening(t *testing.T) {
	node1 := Node{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		NodeNumber:      1,
		DeviceType:      "unknown",
	}

	broadcastingPeriod := 10 * time.Millisecond
	listeningTimeout := broadcastingPeriod * 2

	nb := NewNodeBroadcast(node1, broadcastingPeriod)
	nb.StartBroadcasting()
	defer nb.StopBroadcasting()

	nl := NewNodeListen(node1)
	nl.StartListening()
	defer nl.StartListening()

	timerticker := time.NewTimer(listeningTimeout)
	defer timerticker.Stop()

	select {
	case nodeFound := <-nl.NodesFoundOnNetwork:
		if nodeFound != node1 {
			t.Errorf("Node found on network = %s, expected %s", nodeFound.String(), node1.String())
		}
		timerticker.Stop()
		return
	case <-timerticker.C:
		t.Errorf("Timed out waiting for node to be found on network")
		return
	}
}

// func TestElevatorList(t *testing.T) {
// 	broadcastingPeriod := 100 * time.Millisecond
// 	listeningTimeout := broadcastingPeriod * 30

// 	newNode1 := Node{"sdijfoisj", "0.0.0.0", 9999, 1, "elevator1"}

// 	newNodeB1 := NewNodeBroadcast(newNode1, broadcastingPeriod)
// 	newNodeB1.StartBroadcasting()

// 	newNode2 := Node{"sodicmxzxj", "0.0.0.0", 9999, 2, "elevator2"}

// 	newNodeB2 := NewNodeBroadcast(newNode2, 5*broadcastingPeriod)
// 	newNodeB2.StartBroadcasting()

// 	newNode3 := Node{"klsdlksdldkfjsdi", "0.0.0.0", 9999, 3, "elevator3"}

// 	newNodeB3 := NewNodeBroadcast(newNode3, 10*broadcastingPeriod)
// 	newNodeB3.StartBroadcasting()

// 	newNodeL := NewNodeListen(newNode1)
// 	newNodeL.StartListening()

// 	timerticker := time.NewTimer(listeningTimeout)
// 	defer timerticker.Stop()

// 	select {
// 	case nodeFound := <-nl.NodesFoundOnNetwork:
// 		if nodeFound != node1 {
// 			t.Errorf("Node found on network = %s, expected %s", nodeFound.String(), node1.String())
// 		}
// 		timerticker.Stop()
// 		return
// 	case <-timerticker.C:
// 		t.Errorf("Timed out waiting for node to be found on network")
// 		return
// 	}
// }

func nodeListContains(nl *NodeListen, deviceType string) bool {
	for _, ls := range nl.nodeArray {
		if ls.Node.DeviceType == deviceType {
			return true
		}
	}
	return false
}

func getTimestampFor(nl *NodeListen, deviceType string) time.Time {
	for _, ls := range nl.nodeArray {
		if ls.Node.DeviceType == deviceType {
			return ls.timeSeen
		}
	}
	return time.Time{}
}

// Helper: Count how many times a node with the given device type appears.
func countOccurrences(nl *NodeListen, deviceType string) int {
	count := 0
	for _, ls := range nl.nodeArray {
		if ls.Node.DeviceType == deviceType {
			count++
		}
	}
	return count
}

func TestAddNodeToList(t *testing.T) {
	nl := &NodeListen{
		nodeArray: []LastSeenNode{},
	}

	// 1st test: add one node to the list
	nodeA := Node{"sdijfoisj", "0.0.0.0", 9999, 1, "elevator1"}
	nl.AddNodeToList(nodeA)
	if !nodeListContains(nl, "elevator1") {
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
	nodeB := Node{"sodicmxzxj", "0.0.0.0", 9999, 2, "elevator2"}
	nl.AddNodeToList(nodeB)
	if !nodeListContains(nl, "elevator1") || !nodeListContains(nl, "elevator2") {
		t.Error("Expected both elevator1 and elevator2 to be in the list")
	}

	// 4th test: removing node
	time.Sleep(300 * time.Millisecond) //used to make the node't timestamp bigger, simulating disconnection

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		nl.AddNodeToList(nodeB)
		if !nodeListContains(nl, "elevator1") {
			// if elevator1 is removed, exit the loop.
			break
		}
		if time.Now().After(deadline) {
			// if elevator1 is still there after 500ms give up
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if nodeListContains(nl, "elevator1") {
		t.Error("Expected elevator1 to be removed as stale")
	}
	if !nodeListContains(nl, "elevator2") {
		t.Error("Expected elevator2 to remain in the list")
	}

}
