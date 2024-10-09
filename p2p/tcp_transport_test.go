package p2p

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddr: ":3000",
		Handshake:  NOPHandshakeFunc,
		Decoder:    DefaultDecoder{},
	}
	transport := NewTCPTransport(opts)

	assert.Equal(t, transport.ListenAddr, ":3000")

	assert.Nil(t, transport.ListenAndAccept())
}
