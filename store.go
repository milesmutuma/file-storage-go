package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 3
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]

	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		FileName: hashStr,
	}

}

type PathKey struct {
	PathName string
	FileName string
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")

	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

type PathTransformFunc func(string) PathKey

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key,
	}
}

type StoreOpts struct {
	// Root is the folder name of the root, containing all the files
	Root                  string
	PathTransformPathFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformPathFunc == nil {
		opts.PathTransformPathFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

//	func (s *Store) Has(key string) bool{
//		pathKey := s.PathTransformPathFunc(key)
//
//		os.
//	}

func (s *Store) folderNameWithRoot(pathKey PathKey) string {
	return fmt.Sprintf("%s/%s", s.Root, pathKey.PathName)
}

func (s *Store) fullPathWithRoot(pathKey PathKey) string {
	return fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
}

func (s *Store) firstPathNameWithRoot(key PathKey) string {
	return fmt.Sprintf("%s/%s", s.Root, key.FirstPathName())
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformPathFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.FullPath())
	}()
	return os.RemoveAll(s.firstPathNameWithRoot(pathKey))
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	return buf, err
}
func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformPathFunc(key)

	return os.Open(s.fullPathWithRoot(pathKey))
}
func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformPathFunc(key)
	// make dir
	if err := os.MkdirAll(s.folderNameWithRoot(pathKey), os.ModePerm); err != nil {
		return err
	}

	//buf := new(bytes.Buffer)
	//io.Copy(buf, r)
	//filenameBytes := md5.Sum(buf.Bytes())
	//fileName := hex.EncodeToString(filenameBytes[:])
	pathAndFileName := s.fullPathWithRoot(pathKey)
	f, err := os.Create(pathAndFileName)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("Written (%d) bytes to disk: %s", n, pathAndFileName)

	return nil
}
