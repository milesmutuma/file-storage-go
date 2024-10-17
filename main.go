package main

import (
	"bytes"
	"distributed_file_storage/p2p"
	"log"
	"time"
)

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

	time.Sleep(2 * time.Second)

	go s2.Start()
	time.Sleep(2 * time.Second)

	data := bytes.NewReader([]byte("my big data file here"))
	s2.Store("myprivatedata", data)

	//_, err := s2.Get("myprivatedata")
	//if err != nil {
	//	return
	//}
	//go func() {
	//	for {
	//		msg := <-tr.Consume()
	//		log.Printf("Received message from %s: %s\n", msg.From, string(msg.DataMessage))
	//	}
	//}()
	//
	//if err := tr.ListenAndAccept(); err != nil {
	//	log.Fatal(err)
	//}
	//
	select {}
}
