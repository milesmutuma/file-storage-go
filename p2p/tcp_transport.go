package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	// The underlying connection of the peer. Which in this case is a TCP connection
	net.Conn
	// if we dial and retrieve conn =>> outbound == true
	// if we accept and retrieve conn ==> outbound === false
	outbound bool

	wg *sync.WaitGroup
}

// Send Implement the Peer interface for TCPPeer
func (p *TCPPeer) Send(b []byte) error {
	// Write the data to the underlying connection
	_, err := p.Write(b)
	if err != nil {
		return fmt.Errorf("failed to send data : %w", err)
	}

	return nil
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
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

// Addr implements Transport interface the address
// the transport is acc
func (t *TCPTransport) Addr() string {
	return t.listener.Addr().String()
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

	for {
		msg := RPC{}
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Printf("Error decoding message: %s\n", err)
			continue
		}
		msg.From = conn.RemoteAddr()

		if msg.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}
		t.rpChan <- msg
		fmt.Printf("Received RPC : %+v\n", msg)
	}
}
