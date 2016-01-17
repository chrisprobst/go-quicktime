package quicktime

import (
	"bytes"
	"errors"
	"io"
)

type IsoBmffInitSegment struct {
	FTYP *Atom
	MOOV *Atom
}

type IsoBmffMediaSegment struct {
	MOOF *Atom
	MDAT *Atom
}

type IsoBmffMergedSegment struct {
	*IsoBmffInitSegment
	*IsoBmffMediaSegment
	Buffer []byte
}

func ReadIsoBmffInitSegment(r io.Reader) (*IsoBmffInitSegment, error) {
	ftyp, err := ReadAtom(r)
	if err != nil {
		return nil, err
	}

	if ftyp.Header.Type != "ftyp" {
		return nil, errors.New("FTYP atom expected but got: " + ftyp.Header.Type)
	}

	moov, err := ReadAtom(r)
	if err != nil {
		return nil, err
	}

	if moov.Header.Type != "moov" {
		return nil, errors.New("MOOV atom expected but got: " + moov.Header.Type)
	}

	return &IsoBmffInitSegment{ftyp, moov}, nil
}

func ReadIsoBmffMediaSegment(r io.Reader) (*IsoBmffMediaSegment, error) {
	moof, err := ReadAtom(r)
	if err != nil {
		return nil, err
	}

	if moof.Header.Type != "moof" {
		return nil, errors.New("MOOF atom expected but got: " + moof.Header.Type)
	}

	mdat, err := ReadAtom(r)
	if err != nil {
		return nil, err
	}

	if mdat.Header.Type != "mdat" {
		return nil, errors.New("MDAT atom expected but got: " + mdat.Header.Type)
	}

	return &IsoBmffMediaSegment{moof, mdat}, nil
}

func ReadIsoBmffMergedSegment(r io.Reader, prev *IsoBmffMergedSegment) (*IsoBmffMergedSegment, error) {
	var (
		init  *IsoBmffInitSegment
		media *IsoBmffMediaSegment
		err   error
	)

	// Read init segment
	if prev == nil {
		init, err = ReadIsoBmffInitSegment(r)
		if err != nil {
			return nil, err
		}
	} else {
		init = prev.IsoBmffInitSegment
	}

	// Read media segment
	media, err = ReadIsoBmffMediaSegment(r)
	if err != nil {
		return nil, err
	}

	// Prepare for creating a buffer
	ftyp := init.FTYP.Buffer
	moov := init.MOOV.Buffer
	moof := media.MOOF.Buffer
	mdat := media.MDAT.Buffer
	size := len(ftyp) + len(moov) + len(moof) + len(mdat)
	buffer := bytes.NewBuffer(make([]byte, size))

	// Write all segment one after the other
	buffer.Write(ftyp)
	buffer.Write(moov)
	buffer.Write(moof)
	buffer.Write(mdat)

	return &IsoBmffMergedSegment{init, media, buffer.Bytes()}, nil
}
