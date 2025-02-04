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

	newNode := node.Node{gitHash, "0.0.0.0", 9999, 1, "unknown"}

	newNodeB := node.NewNodeBroadcast(newNode, 1000*time.Millisecond)
	newNodeB.StartBroadcasting()

	newNodeL := node.NewNodeListen(newNode)
	newNodeL.StartListening()

	for {
		select {
		case n := <-newNodeL.NodesFoundOnNetwork:
			Logger.Info().Msgf("Node found on network: %v", n.String())
		}
	}

}
