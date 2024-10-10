package main

import (
	"distributed_file_storage/p2p"
	"log"
)

//func OnPeer(peer p2p.Peer) error {
//	fmt.Println("Doing some logic with the peer outside of TCPTransport")
//	err := peer.Close()
//	if err != nil {
//		return err
//	}
//	return nil
//}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	trOpts := p2p.TCPTransportOpts{
		ListenAddr: listenAddr,
		Handshake:  p2p.NOPHandshakeFunc,
		Decoder:    p2p.DefaultDecoder{},
	}

	tr := p2p.NewTCPTransport(trOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		Transport:         tr,
		PathTransformFunc: CASPathTransformFunc,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)
	tr.OnPeer = s.OnPeer
	return s
}
func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		if err := s1.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	s2.Start()

	//go func() {
	//	for {
	//		msg := <-tr.Consume()
	//		log.Printf("Received message from %s: %s\n", msg.From, string(msg.Payload))
	//	}
	//}()
	//
	//if err := tr.ListenAndAccept(); err != nil {
	//	log.Fatal(err)
	//}
	//
	//select {}
}
