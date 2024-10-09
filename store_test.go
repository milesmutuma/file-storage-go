package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestCASPathTransformFunc(t *testing.T) {
	key := "mombestpicture"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalKey := "cf5d4b01c4d9438c22c56c832f83bd3e8c6304f9"
	expectedPathName := "cf5/d4b/01c/4d9/438/c22/c56/c83/2f8/3bd/3e8/c63/04f"
	if pathKey.PathName != expectedPathName {
		t.Errorf("have %s want %s", pathKey.PathName, expectedPathName)
	}
	if pathKey.FileName != expectedOriginalKey {
		t.Errorf("have %s want %s", pathKey.FileName, expectedOriginalKey)
	}
	//fmt.Println(pathname)
}
func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformPathFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "mombestpicture"
	data := []byte("some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	fmt.Println(string(b))
	if !bytes.Equal(data, b) {
		t.Error("expected data to be the same")
	}
}

func TestStore_Delete(t *testing.T) {
	opts := StoreOpts{
		PathTransformPathFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "mombestpicture"
	data := []byte("some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}
