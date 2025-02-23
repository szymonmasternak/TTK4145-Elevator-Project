package main

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/node"
)

func main() {
	Logger := logger.GetLogger()
	gitHash := elevutils.GetGitHash()
	elevutils.ProcessCmdArgs()
	//var nodeArray []LastSeenNode

	newNode := node.Node{gitHash, "0.0.0.0", 9999, 1, "elevator1"}

	newNodeB := node.NewNodeBroadcast(newNode, 1000*time.Millisecond)
	newNodeB.StartBroadcasting()

	newNode2 := node.Node{gitHash, "0.0.0.0", 9999, 2, "elevator2"}

	newNodeB2 := node.NewNodeBroadcast(newNode2, 7000*time.Millisecond)
	newNodeB2.StartBroadcasting()

	newNode3 := node.Node{gitHash, "0.0.0.0", 9999, 3, "elevator3"}

	newNodeB3 := node.NewNodeBroadcast(newNode3, 5000*time.Millisecond)
	newNodeB3.StartBroadcasting()

	newNodeL := node.NewNodeListen(newNode)
	newNodeL.StartListening()

	for {
		select {
		case n := <-newNodeL.NodesFoundOnNetwork:
			newNodeL.AddNodeToList(n)
			Logger.Info().Msgf("Node found on network: %v", n.String())
		}
	}

}
