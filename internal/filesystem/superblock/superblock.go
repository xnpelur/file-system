package superblock

import (
	"encoding/binary"
	"file-system/internal/filesystem/inode"
	"os"
	"unsafe"
)

type Superblock struct {
	MagicNumber    uint16
	BlockCount     uint32
	InodeCount     uint32
	FreeBlockCount uint32
	FreeInodeCount uint32
	BlockSize      uint32
	InodeSize      uint32
	file           *os.File
}

func (s Superblock) Size() uint32 {
	return uint32(
		unsafe.Sizeof(s.MagicNumber) +
			unsafe.Sizeof(s.BlockCount) +
			unsafe.Sizeof(s.InodeCount) +
			unsafe.Sizeof(s.FreeBlockCount) +
			unsafe.Sizeof(s.FreeInodeCount) +
			unsafe.Sizeof(s.BlockSize) +
			unsafe.Sizeof(s.InodeSize),
	)
}

func NewSuperblock(filesystemSizeInBytes, blockSize uint32, file *os.File) *Superblock {
	s := Superblock{}

	blockCount := filesystemSizeInBytes / blockSize

	s.MagicNumber = 0x1234
	s.BlockCount = blockCount
	s.InodeCount = blockCount
	s.FreeBlockCount = blockCount
	s.FreeInodeCount = blockCount
	s.BlockSize = blockSize
	s.InodeSize = inode.GetInodeSize()
	s.file = file

	return &s
}

func ReadSuperblockAt(file *os.File, offset uint32) (*Superblock, error) {
	data := make([]byte, Superblock{}.Size())

	_, err := file.ReadAt(data, int64(offset))
	if err != nil {
		return nil, err
	}

	s := decodeSuperblock(data)
	s.file = file

	return s, nil
}

func decodeSuperblock(data []byte) *Superblock {
	s := Superblock{}

	s.MagicNumber = binary.BigEndian.Uint16(data[0:2])
	s.BlockCount = binary.BigEndian.Uint32(data[2:6])
	s.InodeCount = binary.BigEndian.Uint32(data[6:10])
	s.FreeBlockCount = binary.BigEndian.Uint32(data[10:14])
	s.FreeInodeCount = binary.BigEndian.Uint32(data[14:18])
	s.BlockSize = binary.BigEndian.Uint32(data[18:22])
	s.InodeSize = binary.BigEndian.Uint32(data[22:26])

	return &s
}

func (s Superblock) Save() error {
	return s.writeAt(s.file, 0)
}

func (s Superblock) writeAt(file *os.File, offset uint32) error {
	data := encodeSuperblock(s)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func encodeSuperblock(value Superblock) []byte {
	size := value.Size()
	data := make([]byte, size)

	binary.BigEndian.PutUint16(data[0:2], value.MagicNumber)
	binary.BigEndian.PutUint32(data[2:6], value.BlockCount)
	binary.BigEndian.PutUint32(data[6:10], value.InodeCount)
	binary.BigEndian.PutUint32(data[10:14], value.FreeBlockCount)
	binary.BigEndian.PutUint32(data[14:18], value.FreeInodeCount)
	binary.BigEndian.PutUint32(data[18:22], value.BlockSize)
	binary.BigEndian.PutUint32(data[22:26], value.InodeSize)

	return data
}
