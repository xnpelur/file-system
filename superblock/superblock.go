package superblock

import (
	"encoding/binary"
	"fmt"
	"os"
)

type Superblock struct {
	magicNumber    uint16 `binary:"big"`
	blockCount     uint32 `binary:"big"`
	inodeCount     uint32 `binary:"big"`
	freeBlockCount uint32 `binary:"big"`
	freeInodeCount uint32 `binary:"big"`
	blockSize      uint32 `binary:"big"`
	inodeSize      uint32 `binary:"big"`
}

func NewSuperblock(blockCount, inodeCount, freeBlockCount, freeInodeCount, blockSize, inodeSize uint32) Superblock {
	s := Superblock{}

	s.magicNumber = 0x1234
	s.blockCount = blockCount
	s.inodeCount = inodeCount
	s.freeBlockCount = freeBlockCount
	s.freeInodeCount = freeInodeCount
	s.blockSize = blockSize
	s.inodeSize = inodeSize

	return s
}

func WriteSuperBlockToFile(file *os.File, offset int, value Superblock) error {
	data := encodeSuperblock(value)
	fmt.Println(data)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func encodeSuperblock(value Superblock) []byte {
	data := make([]byte, binary.Size(value))

	binary.BigEndian.PutUint16(data[0:2], value.magicNumber)
	binary.BigEndian.PutUint32(data[2:6], value.blockCount)
	binary.BigEndian.PutUint32(data[6:10], value.inodeCount)
	binary.BigEndian.PutUint32(data[10:14], value.freeBlockCount)
	binary.BigEndian.PutUint32(data[14:18], value.freeInodeCount)
	binary.BigEndian.PutUint32(data[18:22], value.blockSize)
	binary.BigEndian.PutUint32(data[22:26], value.inodeSize)

	return data
}
