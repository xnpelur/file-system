package superblock

import (
	"encoding/binary"
	"file-system/internal/filesystem/inode"
	"file-system/internal/utils"
	"os"
)

type Superblock struct {
	MagicNumber    uint16
	BlockCount     uint32
	InodeCount     uint32
	FreeBlockCount uint32
	FreeInodeCount uint32
	BlockSize      uint32
	InodeSize      uint32
}

func NewSuperblock(filesystemSizeInBytes, blockSize uint32) *Superblock {
	s := Superblock{}

	blockCount := filesystemSizeInBytes / blockSize

	s.MagicNumber = 0x1234
	s.BlockCount = blockCount
	s.InodeCount = blockCount
	s.FreeBlockCount = blockCount
	s.FreeInodeCount = blockCount
	s.BlockSize = blockSize
	s.InodeSize = uint32(inode.GetInodeSize())

	return &s
}

func (s Superblock) Size() int {
	size, _ := utils.CalculateStructSize(s)
	return size
}

func (s Superblock) WriteToFile(file *os.File, offset int) error {
	data := encodeSuperblock(s)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func encodeSuperblock(value Superblock) []byte {
	size, _ := utils.CalculateStructSize(value)
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
