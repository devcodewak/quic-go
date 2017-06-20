package frames

import "github.com/phuslu/quic-go/protocol"

// AckRange is an ACK range
type AckRange struct {
	FirstPacketNumber protocol.PacketNumber
	LastPacketNumber  protocol.PacketNumber
}
