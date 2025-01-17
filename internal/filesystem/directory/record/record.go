package record

import (
	"encoding/binary"
)

type Record struct {
	Inode        uint32
	RecordLength uint16
	NameLength   uint8
	Name         string
}

func NewRecord(inode uint32, name string) Record {
	recordInstance := Record{}

	recordInstance.Inode = inode
	recordInstance.RecordLength = uint16(len(name)) + 4 + 2 + 1
	recordInstance.NameLength = uint8(len(name))
	recordInstance.Name = name

	return recordInstance
}

func (value Record) Encode() []byte {
	data := make([]byte, value.RecordLength)

	binary.BigEndian.PutUint32(data[0:4], value.Inode)
	binary.BigEndian.PutUint16(data[4:6], value.RecordLength)
	data[6] = value.NameLength
	copy(data[7:7+value.NameLength], value.Name)

	return data
}
