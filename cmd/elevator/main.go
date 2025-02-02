package main

import (
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	_ "github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/nodebroadcast"
)

func main() {
	Log := logger.GetLogger()
	gitHash := elevutils.GetGitHash()

	Log.Info().Msgf("Git Hash: %s", gitHash)

	node := nodebroadcast.NewNodeBroadcast(gitHash, "127.0.0.1", 20005, 1, "master")
	node.StartBroadcasting()

	select {}
}
