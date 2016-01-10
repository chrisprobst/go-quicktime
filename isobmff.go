package quicktime

import (
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
