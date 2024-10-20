package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "supernetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashedStr := hex.EncodeToString(hash[:])

	blocksize := 5
	slicelen := len(hashedStr) / blocksize

	paths := make([]string, slicelen)
	for i := range slicelen {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashedStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashedStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) FirstPathname() string {
	paths := strings.Split(p.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

type StorageOptions struct {
	/*
		Root is the folder name of the root,
		containing all the folders / files of the system
	*/
	Root              string
	PathTransformFunc PathTransformFunc
}

type Storage struct {
	StorageOptions
}

func NewStorage(options StorageOptions) *Storage {
	if options.PathTransformFunc == nil {
		options.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(options.Root) == 0 {
		options.Root = defaultRootFolderName
	}

	return &Storage{
		StorageOptions: options,
	}
}

func (store *Storage) Has(key string) bool {
	pathKey := store.PathTransformFunc(key)

	fullPathWithRoot := fmt.Sprintf("%s/%s", store.Root, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Storage) Clear() error {
	return os.RemoveAll(s.Root)
}

func (store *Storage) Delete(key string) error {
	pathKey := store.PathTransformFunc(key)

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.Filename)
	}()

	firstPathnameWithRoot := fmt.Sprintf("%s/%s", store.Root, pathKey.FirstPathname())

	return os.RemoveAll(firstPathnameWithRoot)
}

func (store *Storage) Write(key string, r io.Reader) (int64, error) {
	return store.writeStream(key, r)
}

func (store *Storage) Read(key string) (int64, io.Reader, error) {
	return store.readStream(key)
}

func (store *Storage) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := store.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", store.Root, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (store *Storage) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := store.PathTransformFunc(key)

	pathnameWithRoot := fmt.Sprintf("%s/%s", store.Root, pathKey.Pathname)
	if err := os.MkdirAll(pathnameWithRoot, os.ModePerm); err != nil {
		return 0, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s", store.Root, pathKey.FullPath())

	file, err := os.Create(fullPathWithRoot)
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(file, r)
	if err != nil {
		return 0, err
	}

	return n, nil
}
