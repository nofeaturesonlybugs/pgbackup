package pgbackup

import (
	"crypto/sha512"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// File is a wrapper around a filepath to expose functionality.
type File string

// SHA512 computes a sha512 hash and writes it to the same path by replacing the
// existing extension with .sha512.
func (f File) SHA512() error {
	var dfd, sfd *os.File
	var err error
	//
	fstr := string(f)
	//
	if sfd, err = os.Open(fstr); err != nil {
		return err
	}
	defer sfd.Close()
	//
	h := sha512.New()
	if _, err = io.Copy(h, sfd); err != nil {
		return err
	}
	//
	hfile := strings.Replace(fstr, filepath.Ext(fstr), ".sha512", 1)
	if dfd, err = os.Create(hfile); err != nil {
		return err
	}
	defer dfd.Close()
	if _, err = fmt.Fprintf(dfd, "%x", h.Sum(nil)); err != nil {
		return err
	}
	//
	return nil
}

// Split splits a file into sequentially numbered chunks of equal size until the last chunk which
// will hold whatever remains.
//
// If dst is empty string then the chunk file names are derived from the filename.  Otherwise
// the part numbers are appended to dst as needed.
func (f File) Split(basepath string, size int, suffixLen int) error {
	fstr := string(f)
	if basepath == "" {
		ext := filepath.Ext(fstr)
		basepath = strings.TrimSuffix(fstr, ext)
	}
	//
	split := &SplitWriter{
		Basepath:     basepath,
		SplitSize:    size,
		SuffixLength: suffixLen,
	}
	defer split.Close()
	//
	sfd, err := os.Open(fstr)
	if err != nil {
		return err
	}
	defer sfd.Close()
	//
	if _, err = io.Copy(split, sfd); err != nil {
		return err
	}
	//
	if err = split.Close(); err != nil {
		return err
	}
	return nil
}
