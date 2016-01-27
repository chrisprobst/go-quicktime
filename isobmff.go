package quicktime

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var ErrFinalSegment = errors.New("MFRA atom read")

type IsoBmffInitSegment struct {
	FTYP *Atom
	MOOV *Atom
}

type IsoBmffMediaSegment struct {
	MOOF                     *Atom
	MDAT                     *Atom
	BaseVideoMediaDecodeTime uint64
	BaseAudioMediaDecodeTime uint64
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

	if moof.Header.Type == "mfra" {
		return nil, ErrFinalSegment
	}

	if moof.Header.Type != "moof" {
		return nil, errors.New("MOOF atom expected but got: " + moof.Header.Type)
	}

	////////////////////////////////////////////////////////////////////////
	//////////////////////////// Base media decode time ////////////////////
	////////////////////////////////////////////////////////////////////////

	// MFHD [skip]
	mfhd, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength:])
	if err != nil {
		return nil, err
	}

	if mfhd.Type != "mfhd" {
		return nil, errors.New("MFHD atom expected but got: " + mfhd.Type)
	}

	// TRAF
	videoTraf, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size:])
	if err != nil {
		return nil, err
	}

	if videoTraf.Type != "traf" {
		return nil, errors.New("TRAF atom expected but got: " + videoTraf.Type)
	}

	// TRAF->TFHD [skip]
	videoTfhd, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size+AtomHeaderLength:])
	if err != nil {
		return nil, err
	}

	if videoTfhd.Type != "tfhd" {
		return nil, errors.New("TFHD atom expected but got: " + videoTfhd.Type)
	}

	// TRAF->TFDT [Base media decode time]
	videoTfdt, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size+AtomHeaderLength+videoTfhd.Size:])
	if err != nil {
		return nil, err
	}

	if videoTfdt.Type != "tfdt" {
		return nil, errors.New("TFDT atom expected but got: " + videoTfdt.Type)
	}

	var baseVideoMediaDecodeTime uint64
	binary.Read(
		bytes.NewReader(moof.Buffer[AtomHeaderLength+mfhd.Size+AtomHeaderLength+videoTfhd.Size+AtomHeaderLength+4:]),
		binary.BigEndian,
		&baseVideoMediaDecodeTime)

	////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////

	// MFHD [skip]
	mfhd, err = ParseAtomHeader(moof.Buffer[AtomHeaderLength:])
	if err != nil {
		return nil, err
	}

	if mfhd.Type != "mfhd" {
		return nil, errors.New("MFHD atom expected but got: " + mfhd.Type)
	}

	// TRAF [skip]
	videoTraf, err = ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size:])
	if err != nil {
		return nil, err
	}

	if videoTraf.Type != "traf" {
		return nil, errors.New("TRAF atom expected but got: " + videoTraf.Type)
	}

	////////////////////////////////////////////////////////////////////////

	// TRAF
	audioTraf, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size+videoTraf.Size:])
	if err != nil {
		return nil, err
	}

	if audioTraf.Type != "traf" {
		return nil, errors.New("TRAF atom expected but got: " + audioTraf.Type)
	}

	// TRAF->TFHD [skip]
	audioTfhd, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size+videoTraf.Size+AtomHeaderLength:])
	if err != nil {
		return nil, err
	}

	if audioTfhd.Type != "tfhd" {
		return nil, errors.New("TFHD atom expected but got: " + audioTfhd.Type)
	}

	// TRAF->TFDT [Base media decode time]
	audioTfdt, err := ParseAtomHeader(moof.Buffer[AtomHeaderLength+mfhd.Size+videoTraf.Size+AtomHeaderLength+audioTfhd.Size:])
	if err != nil {
		return nil, err
	}

	if audioTfdt.Type != "tfdt" {
		return nil, errors.New("TFDT atom expected but got: " + audioTfdt.Type)
	}

	var baseAudioMediaDecodeTime uint64
	binary.Read(
		bytes.NewReader(moof.Buffer[AtomHeaderLength+mfhd.Size+videoTraf.Size+AtomHeaderLength+audioTfhd.Size+AtomHeaderLength+4:]),
		binary.BigEndian,
		&baseAudioMediaDecodeTime)

	////////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////////

	mdat, err := ReadAtom(r)
	if err != nil {
		return nil, err
	}

	if mdat.Header.Type != "mdat" {
		return nil, errors.New("MDAT atom expected but got: " + mdat.Header.Type)
	}

	return &IsoBmffMediaSegment{moof, mdat, baseVideoMediaDecodeTime, baseAudioMediaDecodeTime}, nil
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
	buffer := bytes.NewBuffer(make([]byte, 0, size))

	// Write all segment one after the other
	buffer.Write(ftyp)
	buffer.Write(moov)
	buffer.Write(moof)
	buffer.Write(mdat)

	return &IsoBmffMergedSegment{init, media, buffer.Bytes()}, nil
}
