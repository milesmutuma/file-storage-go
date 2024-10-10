package main

import (
	"distributed_file_storage/p2p"
	"fmt"
	"log"
	"sync"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	p2p.Transport
	BootstrapNodes []string
}

type FileServer struct {
	FileServerOpts
	storage *Store
	quitch  chan struct{}

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:                  opts.StorageRoot,
		PathTransformPathFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		storage:        NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	fmt.Printf("connected with  peer %s: ", p.RemoteAddr().String())
	return nil
}
func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Close()
	}()
	for {
		select {
		case msg := <-s.Consume():
			fmt.Println(msg)
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) bootstrapNodes() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Println("Attempting to connect to " + addr)
			if err := s.Dial(addr); err != nil {
				log.Println("Dial error: ", err)
			}
			//fmt.Println("Connected to " + addr)
		}(addr)
	}

	return nil
}
func (s *FileServer) Start() error {
	if err := s.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNodes()
	s.loop()
	return nil
}
