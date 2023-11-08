package inode

import (
	"encoding/binary"
	"file-system/internal/utils"
	"io"
	"os"
)

type Inode struct {
	TypeAndPermissions uint16
	UserId             uint16
	GroupId            uint16
	FileSize           uint32
	CreationTime       uint32
	ModificationTime   uint32
	LinkCount          uint16
	Blocks             [12]uint32
}

type TypeAndPermissions struct {
	IsFile             bool
	OwnerReadAccess    bool
	OwnerWriteAccess   bool
	OwnerExecuteAccess bool
	GroupReadAccess    bool
	GroupWriteAccess   bool
	GroupExecuteAccess bool
	UsersReadAccess    bool
	UsersWriteAccess   bool
	UsersExecuteAccess bool
}

func NewInode(
	isFile bool,
	numericPermissions int,
	userId int,
	groupId int,
	dataBlocks []uint32,
) *Inode {
	var blocks [12]uint32
	for i, dataBlock := range dataBlocks {
		if i >= 12 {
			break
		}
		blocks[i] = dataBlock
	}
	return &Inode{
		TypeAndPermissions: PackTypeAndPermissions(NewTypeAndPermissions(isFile, numericPermissions)),
		UserId:             uint16(userId),
		GroupId:            uint16(groupId),
		FileSize:           uint32(len(dataBlocks)),
		Blocks:             blocks,
	}
}

func NewTypeAndPermissions(isFile bool, numericPermissions int) TypeAndPermissions {
	users := numericPermissions % 10
	group := numericPermissions / 10 % 10
	owner := numericPermissions / 100 % 10
	return TypeAndPermissions{
		IsFile:             isFile,
		OwnerReadAccess:    (owner>>2)&1 == 1,
		OwnerWriteAccess:   (owner>>1)&1 == 1,
		OwnerExecuteAccess: (owner>>0)&1 == 1,
		GroupReadAccess:    (group>>2)&1 == 1,
		GroupWriteAccess:   (group>>1)&1 == 1,
		GroupExecuteAccess: (group>>0)&1 == 1,
		UsersReadAccess:    (users>>2)&1 == 1,
		UsersWriteAccess:   (users>>1)&1 == 1,
		UsersExecuteAccess: (users>>0)&1 == 1,
	}
}

func GetInodeSize() uint32 {
	inodeDummy := Inode{}
	size, _ := utils.CalculateStructSize(inodeDummy)
	return size
}

func UnpackTypeAndPermissions(value uint16) TypeAndPermissions {
	return TypeAndPermissions{
		IsFile:             value&0b1000000000000000 != 0,
		OwnerReadAccess:    value&0b0100000000000000 != 0,
		OwnerWriteAccess:   value&0b0010000000000000 != 0,
		OwnerExecuteAccess: value&0b0001000000000000 != 0,
		GroupReadAccess:    value&0b0000100000000000 != 0,
		GroupWriteAccess:   value&0b0000010000000000 != 0,
		GroupExecuteAccess: value&0b0000001000000000 != 0,
		UsersReadAccess:    value&0b0000000100000000 != 0,
		UsersWriteAccess:   value&0b0000000010000000 != 0,
		UsersExecuteAccess: value&0b0000000001000000 != 0,
	}
}

func PackTypeAndPermissions(typeAndPermissions TypeAndPermissions) uint16 {
	var value uint16
	if typeAndPermissions.IsFile {
		value |= 0b1000000000000000
	}
	if typeAndPermissions.OwnerReadAccess {
		value |= 0b0100000000000000
	}
	if typeAndPermissions.OwnerWriteAccess {
		value |= 0b0010000000000000
	}
	if typeAndPermissions.OwnerExecuteAccess {
		value |= 0b0001000000000000
	}
	if typeAndPermissions.GroupReadAccess {
		value |= 0b0000100000000000
	}
	if typeAndPermissions.GroupWriteAccess {
		value |= 0b0000010000000000
	}
	if typeAndPermissions.GroupExecuteAccess {
		value |= 0b0000001000000000
	}
	if typeAndPermissions.UsersReadAccess {
		value |= 0b0000000100000000
	}
	if typeAndPermissions.UsersWriteAccess {
		value |= 0b0000000010000000
	}
	if typeAndPermissions.UsersExecuteAccess {
		value |= 0b0000000001000000
	}

	return value
}

func ReadInodeAt(file *os.File, offset uint32) (*Inode, error) {
	_, err := file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 68)
	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	return decodeInode(data), nil
}

func decodeInode(data []byte) *Inode {
	inode := Inode{}

	inode.TypeAndPermissions = binary.BigEndian.Uint16(data[0:2])
	inode.UserId = binary.BigEndian.Uint16(data[2:4])
	inode.GroupId = binary.BigEndian.Uint16(data[4:6])
	inode.FileSize = binary.BigEndian.Uint32(data[6:10])
	inode.CreationTime = binary.BigEndian.Uint32(data[10:14])
	inode.ModificationTime = binary.BigEndian.Uint32(data[14:18])
	inode.LinkCount = binary.BigEndian.Uint16(data[18:20])

	for i := 0; i < 12; i++ {
		offset := 20 + i*4
		inode.Blocks[i] = binary.BigEndian.Uint32(data[offset : offset+4])
	}

	return &inode
}

func (inode Inode) WriteAt(file *os.File, offset uint32) error {
	data := inode.encode()

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (value Inode) encode() []byte {
	data := make([]byte, 68)

	binary.BigEndian.PutUint16(data[0:2], value.TypeAndPermissions)
	binary.BigEndian.PutUint16(data[2:4], value.UserId)
	binary.BigEndian.PutUint16(data[4:6], value.GroupId)
	binary.BigEndian.PutUint32(data[6:10], value.FileSize)
	binary.BigEndian.PutUint32(data[10:14], value.CreationTime)
	binary.BigEndian.PutUint32(data[14:18], value.ModificationTime)
	binary.BigEndian.PutUint16(data[18:20], value.LinkCount)

	for i := 0; i < 12; i++ {
		offset := 20 + i*4
		binary.BigEndian.PutUint32(data[offset:offset+4], value.Blocks[i])
	}

	return data
}
