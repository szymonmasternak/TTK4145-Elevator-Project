package elevnet

import (
	"testing"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
)

func TestStartBroadcastingListening(t *testing.T) {
	metaData := elevmetadata.ElevMetaData{
		SoftwareVersion: "smj2acjkvv4h1zkwjz2ocsn2lkfrjmzf9qn4i2m3",
		IpAddress:       "0.0.0.0",
		PortNumber:      9999,
		Identifier:      "uwvvblrtct",
	}

	broadcastingPeriod := 10 * time.Millisecond
	listeningTimeout := broadcastingPeriod * 2

	network := NewElevatorNetwork(&metaData)
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
