package main

import (
	"distributed_file_storage/p2p"
	"fmt"
	"log"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Println("Doing some logic with the peer outside of TCPTransport")
	err := peer.Close()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	trOpts := p2p.TCPTransportOpts{
		ListenAddr: ":3000",
		Handshake:  p2p.NOPHandshakeFunc,
		Decoder:    p2p.DefaultDecoder{},
		OnPeer:     OnPeer,
	}

	tr := p2p.NewTCPTransport(trOpts)

	go func() {
		for {
			msg := <-tr.Consume()
			log.Printf("Received message from %s: %s\n", msg.From, string(msg.Payload))
		}
	}()

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
