package erpel

import (
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// Marker is a position within a file.
type Marker struct {
	Inode  uint64 `json:"inode"`
	Offset int64  `json:"offset"`
}

func getInode(f *os.File) (os.FileInfo, uint64, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, errors.WithMessage(err, f.Name())
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, 0, nil
	}

	return fi, stat.Ino, nil
}

// Position returns a Marker for the given file.
func Position(f *os.File) (Marker, error) {
	pos, err := f.Seek(0, 1)
	if err != nil {
		return Marker{}, errors.WithMessage(err, f.Name())
	}

	_, inode, err := getInode(f)
	if err != nil {
		return Marker{}, errors.WithMessage(err, f.Name())
	}

	m := Marker{
		Offset: pos,
		Inode:  inode,
	}

	return m, nil
}

// isNewFile returns true iff the underlying file has been changed, i.e. a log
// file was moved away and a new file began.
func (m Marker) isNewFile(f *os.File) (bool, error) {
	fi, inode, err := getInode(f)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if m.Inode != 0 {
		if inode != m.Inode {
			return true, nil
		}
	}

	if fi.Size() < m.Offset {
		return true, nil
	}

	return false, nil
}

// Seek moves f to the position of the marker, so that new bytes can be read.
// When the file has been replaced by a new file, calling Seek() does nothing.
func (m Marker) Seek(f *os.File) error {

	// if this is the null marker, do nothing.
	if m.Offset == 0 && m.Inode == 0 {
		return nil
	}

	offset := m.Offset

	newFile, err := m.isNewFile(f)
	if err != nil {
		return err
	}

	if newFile {
		offset = 0
	}

	_, err = f.Seek(offset, 0)
	return err
}
