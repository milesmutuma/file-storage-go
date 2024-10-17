package main

import (
	"bytes"
	"distributed_file_storage/p2p"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
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

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

func (s *FileServer) stream(message *Message) error {
	var peers []io.Writer
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(message)
}

func (s *FileServer) broadcast(message *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(message); err != nil {
		return err
	}

	for _, peer := range s.peers {
		fmt.Println("Sending broadcast message")
		_ = peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

type MessageGetFile struct {
	Key string
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.storage.Has(key) {
		fmt.Printf("[%s] serving file (%s) from local disk", s.Addr(), key)
		return s.storage.Read(key)
	}

	fmt.Printf("Don't have %s file locally, searching the network", key)
	message := Message{
		Payload: MessageGetFile{Key: key},
	}

	if err := s.broadcast(&message); err != nil {
		return nil, err
	}

	time.Sleep(3 * time.Millisecond)

	for _, peer := range s.peers {
		filebuffer := new(bytes.Buffer)
		n, err := io.CopyN(filebuffer, peer, 22)
		if err != nil {
			return nil, err
		}
		fmt.Printf("[%s] received %d bytes over network from (%s)", s.Transport.Addr(), n, peer.RemoteAddr())
		fmt.Println(filebuffer.String())

		peer.CloseStream()
	}

	select {}
}

func (s *FileServer) Store(key string, r io.Reader) error {
	// 1. store this file to disk
	// 2. broadcast this file to all known peers
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.storage.Write(key, tee)
	if err != nil {
		return err
	}

	message := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	err = s.broadcast(&message)
	if err != nil {
		return err
	}

	fmt.Println("Finished sending broadcast message")

	time.Sleep(300 * time.Millisecond)

	for _, peer := range s.peers {
		err := peer.Send([]byte{p2p.IncomingStream})
		fmt.Println(fileBuffer)
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		fmt.Printf("Wrote (%d) bytes to peer", n)
	}
	return nil
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	fmt.Printf("connected with  peer %s:\n", p.RemoteAddr().String())
	return nil
}
func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Close()
	}()
	for {
		select {
		case rpc := <-s.Consume():
			var message Message
			fmt.Println("Hello")
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&message); err != nil {
				fmt.Println("decoding error: ", err)
			}
			if err := s.handleMessage(rpc.From.String(), &message); err != nil {
				fmt.Printf("handle message error %s", err)
			}

			//peer, ok := s.peers[rpc.From.String()]
			//if !ok {
			//	panic("peer not found")
			//}
			//
			//b := make([]byte, 1000)
			//if _, err := peer.Read(b); err != nil {
			//	panic("peer read error")
			//}
			//
			//fmt.Printf("received message from peer %v", peer.RemoteAddr().String())
			//
			//peer.(*p2p.TCPPeer).wg.Done()
			//if err := s.handleMessage(&message); err != nil {
			//	log.Println("Error handling message:", err)
			//	continue
			//}

		case <-s.quitch:
			return
		}
	}
}

func (f *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return f.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return f.handleMessageGetFile(from, v)
	}

	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %x not found in the peer list", from)
	}
	n, err := s.storage.Write(msg.Key, io.LimitReader(peer, msg.Size))

	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk", s.Addr(), n)
	peer.CloseStream()

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	fmt.Println("sasa")
	if !s.storage.Has(msg.Key) {
		return fmt.Errorf("file %s does not exist on disk", msg.Key)
	}

	reader, err := s.storage.Read(msg.Key)
	if err != nil {
		return err
	}
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %x not found in the peer list", from)
	}

	_ = peer.Send([]byte{p2p.IncomingStream})
	n, err := io.Copy(peer, reader)
	if err != nil {
		return err
	}

	fmt.Printf("Sent (%d) bytes to peer %s", n, from)

	return nil
}
func (f *FileServer) Stop() {
	close(f.quitch)
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

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
