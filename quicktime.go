package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
)

const AtomHeaderLength = 8

type AtomHeader struct {
	Size     int
	DataSize int
	Type     string
}

type Atom struct {
	Header *AtomHeader
	Buffer []byte
}

func ParseAtomHeader(buffer []byte) (*AtomHeader, error) {
	if len(buffer) < AtomHeaderLength {
		return nil, errors.New("Invalid buffer size")
	}

	// Read atom size
	var atomSize uint32
	err := binary.Read(bytes.NewReader(buffer), binary.BigEndian, &atomSize)
	if err != nil {
		return nil, err
	}

	if atomSize == 0 {
		return nil, errors.New("Zero size not supported yet")
	}

	if atomSize == 1 {
		return nil, errors.New("64 bit atom size not supported yet")
	}

	// Read atom type
	atomType := string(buffer[4:8])

	return &AtomHeader{int(atomSize), int(atomSize) - AtomHeaderLength, atomType}, nil
}

func ReadAtom(r io.Reader) (*Atom, error) {
	var atomBuffer bytes.Buffer

	// Read header
	_, err := io.CopyN(&atomBuffer, r, AtomHeaderLength)
	if err != nil {
		return nil, err
	}

	// Parse header
	atomHeader, err := ParseAtomHeader(atomBuffer.Bytes())
	if err != nil {
		return nil, err
	}

	// Read atom data
	_, err = io.CopyN(&atomBuffer, r, int64(atomHeader.DataSize))
	if err != nil {
		return nil, err
	}

	// Create atom
	atom := &Atom{atomHeader, atomBuffer.Bytes()}

	return atom, nil
}

func main() {

	var init *IsoBmffInitSegment
	var err error

	for {
		if init == nil {
			init, err = ReadIsoBmffInitSegment(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Println(init.FTYP.Header, init.MOOV.Header)

		media, err := ReadIsoBmffMediaSegment(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(media.MOOF.Header, media.MDAT.Header)
	}
}
