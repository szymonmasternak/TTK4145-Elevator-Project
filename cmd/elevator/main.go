package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/node"
)

type LastSeenNode struct {
	Node     node.Node
	timeSeen time.Time
}

func main() {
	Logger := logger.GetLogger()
	gitHash := elevutils.GetGitHash()
	elevutils.ProcessCmdArgs()
	var nodeArray []LastSeenNode
	//var index int
	//index = 0

	newNode := node.Node{gitHash, "0.0.0.0", 9999, 1, "unknown"}

	newNodeB := node.NewNodeBroadcast(newNode, 1000*time.Millisecond)
	newNodeB.StartBroadcasting()

	newNode2 := node.Node{gitHash, "0.0.0.0", 9999, 2, "unknown"}

	newNodeB2 := node.NewNodeBroadcast(newNode2, 7000*time.Millisecond)
	newNodeB2.StartBroadcasting()

	newNode3 := node.Node{gitHash, "0.0.0.0", 9999, 3, "unknown"}

	newNodeB3 := node.NewNodeBroadcast(newNode3, 5000*time.Millisecond)
	newNodeB3.StartBroadcasting()

	newNodeL := node.NewNodeListen(newNode)
	newNodeL.StartListening()

	for {
		select {
		case n := <-newNodeL.NodesFoundOnNetwork:
			Logger.Info().Msgf("Node found on network: %v", n.String())

			var repeat bool
			repeat = false
			for i := 0; i < len(nodeArray); i++ {
				if n.NodeNumber == nodeArray[i].Node.NodeNumber {
					repeat = true
					nodeArray[i].timeSeen = time.Now()
				}
			}
			if !repeat {
				nodeArray = append(nodeArray, LastSeenNode{n, time.Now()})
				//index = index + 1
			}
			Logger.Info().Msgf("Node list: ")

			filtered := nodeArray[:0] // Keep only valid elements

			for i := 0; i < len(nodeArray); i++ {
				if nodeArray[i].timeSeen.Second() >= time.Now().Second()-2 {
					filtered = append(filtered, nodeArray[i]) // Keep only non-stale nodes
					s := fmt.Sprintf("%v, ", nodeArray[i].Node.NodeNumber)
					io.WriteString(os.Stdout, s)
				} else {
					Logger.Info().Msg("Node number has become stale, removing from the list")
				}
			}

			nodeArray = filtered // Update original slice

			io.WriteString(os.Stdout, "\n")
		}
	}

}
