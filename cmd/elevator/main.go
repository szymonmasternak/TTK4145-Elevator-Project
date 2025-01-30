package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevutils"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/logger"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/bcast"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/localip"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/libs/Network-go/network/peers"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//
//	will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

func main() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	gitHash := elevutils.GetGitHash()

	Log := logger.GetLogger()
	Log.Info().Msgf("Git Hash: %s", gitHash)

	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	// We make a channel for receiving updates on the id's of the peers that are
	//  alive on the network
	peerUpdateCh := make(chan peers.PeerUpdate)
	// We can disable/enable the transmitter after it has been started.
	// This could be used to signal that we are somehow "unavailable".
	peerTxEnable := make(chan bool)
	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	// We make channels for sending and receiving our custom data types
	helloTx := make(chan HelloMsg)
	helloRx := make(chan HelloMsg)
	// ... and start the transmitter/receiver pair on some port
	// These functions can take any number of channels! It is also possible to
	//  start multiple transmitters/receivers on the same port.
	go bcast.Transmitter(16569, helloTx)
	go bcast.Receiver(16569, helloRx)

	// The example message. We just send one of these every second.
	go func() {
		helloMsg := HelloMsg{"Hello from " + id, 0}
		for {
			helloMsg.Iter++
			helloTx <- helloMsg
			time.Sleep(1 * time.Second)
		}
	}()

	Log.Info().Msg("Started")
	for {
		select {
		case p := <-peerUpdateCh:
			Log.Info().Msgf("Peer update:")
			Log.Info().Msgf("  Peers:    %q", p.Peers)
			Log.Info().Msgf("  New:      %q", p.New)
			Log.Info().Msgf("  Lost:     %q", p.Lost)

		case a := <-helloRx:
			Log.Info().Msgf("Received: %#v", a)
		}
	}
}
