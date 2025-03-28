package elevnet

import (
	"time"

	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevconsts"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevmetadata"
	"github.com/szymonmasternak/TTK4145-Elevator-Project/internal/elevstate"
)

type NetworkPacketType int

const (
	PACKET_TYPE_HEARTBEAT NetworkPacketType = iota
	PACKET_TYPE_STATE
	PACKET_TYPE_ACK
	PACKET_TYPE_DO_REQ
)

type NetworkPacket struct {
	PacketType NetworkPacketType         `json:"packet_type"`
	MetaData   elevmetadata.ElevMetaData `json:"meta_data"`
	Time       time.Time                 `json:"time"`
}

type HeartBeatPacket struct {
	NetworkPacket
}

type StatePacket struct {
	NetworkPacket
	State elevstate.ElevatorState `json:"state"`
}

type ButtonPacket struct {
	NetworkPacket
	Floor  int               `json:"floor"`
	Button elevconsts.Button `json:"button"`
}

type AckPacket struct {
	NetworkPacket
	MessageHash string `json:"message_hash"`
}

type DoRequestPacket struct {
	NetworkPacket
	Floor  int               `json:"floor"`
	Button elevconsts.Button `json:"button"`
}
