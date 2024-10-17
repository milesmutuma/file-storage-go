package p2p

import "net"

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC folds any arbitrary data that us being sent over the network
// each transport between two nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
	Stream  bool
}
