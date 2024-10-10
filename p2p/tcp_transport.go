package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	// conn is the underlying connection of the Peer
	conn net.Conn

	// if we dial and retrieve conn =>> outbound == true
	// if we accept and retrieve conn ==> outbound === false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// Close implements the Peer interface
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

// RemoteAddr implements the Peer interface and will return the
// remote address of its underlying connection
func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}

type TCPTransportOpts struct {
	ListenAddr string
	Handshake  HandshakerFunc
	Decoder    Decoder
	OnPeer     func(peer Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpChan   chan RPC

	mu sync.RWMutex
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpChan:           make(chan RPC),
	}
}

// Consume implements the Transport interface, which will return read-nly channel
// for reading the incoming messages received from another peer in the network connection
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpChan
}

// Close implements the Transport interface, which
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}
func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	fmt.Printf("TCP server listening on %s\n", t.ListenAddr)
	go t.startAndAcceptLoop()

	return nil
}

func (t *TCPTransport) startAndAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}
		go t.handleConn(conn, false)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		fmt.Print("Dropping peer connection")
	}()

	peer := NewTCPPeer(conn, outbound)

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	if err = t.Handshake(peer); err != nil {
		conn.Close()
		if err != nil {
			fmt.Printf("Error while handshaking: %s\n", err)
			return
		}
	}

	// Read Loop
	msg := RPC{}
	for {
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Printf("Error decoding message: %s\n", err)
			continue
		}
		msg.From = conn.RemoteAddr()
		t.rpChan <- msg

		fmt.Printf("Received RPC : %+v\n", msg)
	}
}
