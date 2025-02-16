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
