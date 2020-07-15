package data

import (
	"fmt"
	"io"
)

type Truncater interface {
	Truncate(size int64) error
}

func pos(s io.Seeker) (int64, error) {
	return s.Seek(0, io.SeekCurrent)
}

func length(s io.Seeker) (int64, error) {
	pos, err := pos(s)
	if err != nil {
		return pos, err
	}
	eof, err := s.Seek(0, io.SeekEnd)
	if _, err2 := s.Seek(pos, io.SeekStart); err == nil {
		err = err2
	}
	return eof, err
}

func rewind(s io.Seeker) error {
	_, err := s.Seek(0, io.SeekStart)
	return err
}

func Reorder(rw io.ReadWriteSeeker) error {
	if err := rewind(rw); err != nil {
		return err
	}

	origLength, err := length(rw)
	if err != nil {
		return err
	}

	r := NewReader(rw)
	s := new(Set).Init()

	for r.Scan() {
		s.Add(r.Item().Clone())
	}

	if err := r.Err(); err != nil {
		return err
	}

	if err := rewind(rw); err != nil {
		return err
	}

	w := NewWriter(rw)
	it := s.Iterator()
	for it.Next() {
		if err := w.Write(it.Value()); err != nil {
			return err
		}
	}

	newLength, err := pos(rw)
	if err != nil {
		return err
	}

	if newLength < origLength {
		if t, ok := rw.(Truncater); ok {
			if err := t.Truncate(newLength); err != nil {
				return err
			}
		} else {
			diff := origLength - newLength
			return fmt.Errorf("unable to truncate tail %d bytes from stream", diff)
		}
	}

	return nil
}
