package main

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/nodebroadcast"
)

func main() {
	gitHash := elevutils.GetGitHash()
	elevutils.ProcessCmdArgs()

	node := nodebroadcast.NewNodeBroadcast(gitHash, "127.0.0.1", 20005, 1, "master")
	node.StartBroadcasting()

	select {}
}
